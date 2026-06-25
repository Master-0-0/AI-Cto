package protocol

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeEnvelope(t *testing.T) {
	payload := []byte("hello world encrypted payload data")
	msgType := uint32(MsgHandshakeRequest)

	envelope := EncodeEnvelope(msgType, payload)

	// Verify header
	if len(envelope) < EnvelopeOverhead {
		t.Fatalf("envelope too short: %d bytes", len(envelope))
	}

	decType, decPayload, err := DecodeEnvelope(envelope)
	if err != nil {
		t.Fatalf("DecodeEnvelope failed: %v", err)
	}

	if decType != msgType {
		t.Errorf("message type mismatch: expected %d, got %d", msgType, decType)
	}

	if !bytes.Equal(decPayload, payload) {
		t.Errorf("payload mismatch")
	}
}

func TestReadFullEnvelope(t *testing.T) {
	payload := []byte("test payload data")
	env := EncodeEnvelope(MsgHeartbeat, payload)

	reader := bytes.NewReader(env)
	msgType, decPayload, err := ReadFullEnvelope(reader)
	if err != nil {
		t.Fatalf("ReadFullEnvelope failed: %v", err)
	}

	if msgType != MsgHeartbeat {
		t.Errorf("expected type %d, got %d", MsgHeartbeat, msgType)
	}

	if !bytes.Equal(decPayload, payload) {
		t.Errorf("payload mismatch")
	}
}

func TestDecodeEnvelopeTooShort(t *testing.T) {
	_, _, err := DecodeEnvelope([]byte{0x00, 0x01})
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestDecodeEnvelopeLengthMismatch(t *testing.T) {
	// Declare length 100 but only provide 10 bytes
	data := make([]byte, 10)
	data[0] = 0x00
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 100
	_, _, err := DecodeEnvelope(data)
	if err == nil {
		t.Error("expected length mismatch error")
	}
}

func TestReadFullEnvelopeTooLarge(t *testing.T) {
	// Create envelope with oversized declared length
	data := make([]byte, 8)
	data[0] = 0xFF
	data[1] = 0xFF
	data[2] = 0xFF
	data[3] = 0xFF // ~4GB

	reader := bytes.NewReader(data)
	_, _, err := ReadFullEnvelope(reader)
	if err == nil {
		t.Error("expected size limit error")
	}
}

func TestSerializeDeserialize(t *testing.T) {
	original := HandshakeRequest{
		AgentID:  "test-agent-123",
		Hostname: "test-host",
		OS:       "linux",
		Arch:     "amd64",
		Username: "root",
		PID:      1234,
		LocalIPs: []string{"192.168.1.1", "10.0.0.1"},
		Version:  "1.0.0",
	}

	data, err := Serialize(original)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	var decoded HandshakeRequest
	if err := Deserialize(data, &decoded); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	if decoded.AgentID != original.AgentID {
		t.Errorf("AgentID mismatch: %s vs %s", decoded.AgentID, original.AgentID)
	}
	if decoded.Hostname != original.Hostname {
		t.Errorf("Hostname mismatch")
	}
	if len(decoded.LocalIPs) != 2 {
		t.Errorf("LocalIPs length mismatch: %d", len(decoded.LocalIPs))
	}
}
