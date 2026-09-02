package store

import (
	"crypto/rand"
	"encoding/hex"
)

func shortID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
