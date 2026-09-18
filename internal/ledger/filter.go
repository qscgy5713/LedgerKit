package ledger

import (
	"strings"
	"time"
)

// Filter narrows which postings a report considers.
type Filter struct {
	AccountPattern string
	From           *time.Time
	To             *time.Time
}

func matchAccount(account, pattern string) bool {
	if pattern == "" {
		return true
	}
	return account == pattern || strings.HasPrefix(account, pattern+":")
}

func inRange(d time.Time, f Filter) bool {
	if f.From != nil && d.Before(*f.From) {
		return false
	}
	if f.To != nil && d.After(*f.To) {
		return false
	}
	return true
}
