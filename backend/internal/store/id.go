package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func shortID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// DatedPublicID returns prefix_YYYYMMDD_xxxxxxxxxxxx (12 hex chars).
func DatedPublicID(prefix string) string {
	return fmt.Sprintf("%s_%s_%s", prefix, time.Now().UTC().Format("20060102"), shortID())
}

// NewPublicID keeps short random id for entities that don't need dated format.
func NewPublicID() string {
	return shortID()
}

// NewTaskID / NewLedgerID — platform-facing formal numbers.
func NewTaskID() string   { return DatedPublicID("tsk") }
func NewLedgerID() string { return DatedPublicID("TX") }
func NewUploadID() string { return DatedPublicID("upl") }
func NewRouteID() string  { return "rte_" + shortID() }
