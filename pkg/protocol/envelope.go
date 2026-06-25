package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// EncodeEnvelope builds a complete wire-format message:
// [TotalLength:4][MessageType:4][EncryptedPayload:variable][GCMTag:16]
// The payload is the already-encrypted ciphertext (nonce + ciphertext).
func EncodeEnvelope(msgType uint32, encryptedPayload []byte) []byte {
	totalLen := LengthPrefixSize + TypePrefixSize + len(encryptedPayload)
	buf := make([]byte, totalLen)

	binary.BigEndian.PutUint32(buf[0:4], uint32(totalLen))
	binary.BigEndian.PutUint32(buf[4:8], msgType)
	copy(buf[8:], encryptedPayload)

	return buf
}

// DecodeEnvelope extracts the message type and encrypted payload from a
// complete envelope buffer. The input should be the full envelope including
// the length prefix.
func DecodeEnvelope(data []byte) (msgType uint32, encryptedPayload []byte, err error) {
	if len(data) < EnvelopeOverhead {
		return 0, nil, fmt.Errorf("envelope too short: %d bytes (minimum %d)", len(data), EnvelopeOverhead)
	}

	totalLen := binary.BigEndian.Uint32(data[0:4])
	if int(totalLen) != len(data) {
		return 0, nil, fmt.Errorf("envelope length mismatch: declared %d, actual %d", totalLen, len(data))
	}

	msgType = binary.BigEndian.Uint32(data[4:8])
	encryptedPayload = data[8:]

	return msgType, encryptedPayload, nil
}

// ReadFullEnvelope reads a complete envelope from a reader.
// It first reads the 4-byte length prefix, then reads the remaining bytes.
func ReadFullEnvelope(reader io.Reader) (msgType uint32, encryptedPayload []byte, err error) {
	lenBuf := make([]byte, LengthPrefixSize)
	if _, err := io.ReadFull(reader, lenBuf); err != nil {
		return 0, nil, fmt.Errorf("reading length prefix: %w", err)
	}

	totalLen := binary.BigEndian.Uint32(lenBuf)
	if totalLen < EnvelopeOverhead {
		return 0, nil, fmt.Errorf("invalid declared length: %d", totalLen)
	}
	if totalLen > MaxMessageSize {
		return 0, nil, fmt.Errorf("message too large: %d bytes (max %d)", totalLen, MaxMessageSize)
	}

	remaining := make([]byte, totalLen-LengthPrefixSize)
	if _, err := io.ReadFull(reader, remaining); err != nil {
		return 0, nil, fmt.Errorf("reading envelope body: %w", err)
	}

	msgType = binary.BigEndian.Uint32(remaining[0:4])
	encryptedPayload = remaining[4:]

	return msgType, encryptedPayload, nil
}
