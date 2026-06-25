package protocol

import (
	"bytes"
	"encoding/gob"
)

// Serialize encodes a value using Gob and returns the byte slice.
func Serialize(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Deserialize decodes Gob-encoded data into the provided value.
func Deserialize(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}
