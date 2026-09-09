package ctb

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// HashHolder stores the current CTB hash in a thread-safe manner.
// Shared between the webhook (reads) and the CTB watcher (writes).
type HashHolder struct {
	mu   sync.RWMutex
	hash string
}

func NewHashHolder() *HashHolder {
	return &HashHolder{}
}

func (h *HashHolder) Get() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hash
}

func (h *HashHolder) Set(hash string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hash = hash
}

// ComputeAndSet hashes the given trust bundle content with SHA-256 and stores the result.
func (h *HashHolder) ComputeAndSet(trustBundle string) {
	sum := sha256.Sum256([]byte(trustBundle))
	h.Set(hex.EncodeToString(sum[:]))
}
