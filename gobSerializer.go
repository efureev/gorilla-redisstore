package redisstore

import (
	"bytes"
	"encoding/gob"

	"github.com/gorilla/sessions"
)

// GobSerializer is Gob Serializer
type GobSerializer struct{}

// Serialize to []byte
func (gs GobSerializer) Serialize(s *sessions.Session) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(s.Values)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

// Deserialize back to map[string]interface{}
func (gs GobSerializer) Deserialize(d []byte, s *sessions.Session) error {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	return dec.Decode(&s.Values)
}
