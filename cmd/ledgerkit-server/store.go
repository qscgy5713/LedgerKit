package main

import (
	"os"
	"sync"

	"ledgerkit/internal/journal"
)

// store guards reads and writes to the journal file within this process
// with an RWMutex: concurrent reads (balance/register/check/accounts) don't
// block each other since they only touch the filesystem, not any shared
// in-memory state, while a write takes the exclusive lock. This does not
// protect against another process (e.g. the `ledgerkit` CLI) writing the
// same file concurrently — see README's "known limitations" section.
type store struct {
	mu   sync.RWMutex
	path string
}

func newStore(path string) *store {
	return &store{path: path}
}

func (s *store) load() (*journal.Journal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return journal.LoadFile(s.path)
}

// exists reports whether the journal file is already present, without
// parsing it. Used so POST /api/transactions matches the read endpoints'
// behavior of requiring `ledgerkit init` first, instead of silently
// creating a bare journal (missing the usual header/opening-balance
// bootstrap) at whatever path LEDGERKIT_JOURNAL happens to point at.
func (s *store) exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, err := os.Stat(s.path)
	return err == nil
}

func (s *store) append(t journal.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return journal.AppendTransaction(s.path, t)
}
