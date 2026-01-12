package bmsbridge

import (
	"strings"
	"sync"
	"time"
)

type BoolDelta struct {
	Key      string
	OldValue bool
	NewValue bool
}

type BoolStateStore struct {
	mu       sync.Mutex
	byDevice map[string]map[string]bool
	lastSeen map[string]time.Time
	ttl      time.Duration
}

func NewBoolStateStore(ttl time.Duration) *BoolStateStore {
	return &BoolStateStore{
		byDevice: make(map[string]map[string]bool),
		lastSeen: make(map[string]time.Time),
		ttl:      ttl,
	}
}

func (s *BoolStateStore) Cleanup(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for deviceID, ts := range s.lastSeen {
		if now.Sub(ts) > s.ttl {
			delete(s.lastSeen, deviceID)
			delete(s.byDevice, deviceID)
		}
	}
}

func (s *BoolStateStore) DiffAndSet(deviceID string, flat map[string]any, prefixes []string) []BoolDelta {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.lastSeen[deviceID] = now

	prev := s.byDevice[deviceID]
	if prev == nil {
		prev = make(map[string]bool, 64)
		s.byDevice[deviceID] = prev
	}

	var deltas []BoolDelta
	for k, v := range flat {
		if !matchesAnyPrefix(k, prefixes) {
			continue
		}
		b, ok := v.(bool)
		if !ok {
			continue
		}
		old, hasOld := prev[k]
		if !hasOld || old != b {
			deltas = append(deltas, BoolDelta{Key: k, OldValue: old, NewValue: b})
		}
		prev[k] = b
	}
	return deltas
}

func matchesAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
