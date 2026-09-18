package main

import (
	"fmt"
	"os"
	"time"

	"ledgerkit/internal/journal"
)

const defaultJournalPath = "ledger.journal"

// journalPath resolves the journal file to use: the LEDGERKIT_JOURNAL env
// var if set, otherwise ./ledger.journal.
func journalPath() string {
	if p := os.Getenv("LEDGERKIT_JOURNAL"); p != "" {
		return p
	}
	return defaultJournalPath
}

func loadJournal(path string) (*journal.Journal, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("journal file %q does not exist (run `ledgerkit init` first, or set LEDGERKIT_JOURNAL)", path)
		}
		return nil, err
	}
	defer f.Close()
	return journal.Parse(f)
}

// parseDateFlag parses a "YYYY-MM-DD" flag value; an empty string returns nil.
func parseDateFlag(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q, want YYYY-MM-DD", s)
	}
	return &t, nil
}
