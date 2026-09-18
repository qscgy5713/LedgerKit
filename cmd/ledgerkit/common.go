package main

import (
	"fmt"
	"os"

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
	j, err := journal.LoadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("journal file %q does not exist (run `ledgerkit init` first, or set LEDGERKIT_JOURNAL)", path)
		}
		return nil, err
	}
	return j, nil
}
