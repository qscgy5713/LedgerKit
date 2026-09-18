package ledger

import (
	"sort"
	"time"

	"ledgerkit/internal/journal"
)

// Entry is one line of a register report: a single posting plus the running
// balance (per commodity) of everything matched so far, in date order.
type Entry struct {
	Date        time.Time
	Description string
	Account     string
	Amount      journal.Amount
	Running     journal.Amounts
}

// Register lists matching postings in date order with a running balance.
func Register(txns []journal.Transaction, f Filter) []Entry {
	sorted := make([]journal.Transaction, len(txns))
	copy(sorted, txns)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Date.Before(sorted[j].Date)
	})

	running := journal.NewAmounts()
	var entries []Entry
	for _, t := range sorted {
		if !inRange(t.Date, f) {
			continue
		}
		for _, p := range t.Postings {
			if p.Amount == nil || !matchAccount(p.Account, f.AccountPattern) {
				continue
			}
			running = running.Add(*p.Amount) // Add returns a new value, so this snapshot is safe to keep per-entry
			entries = append(entries, Entry{
				Date:        t.Date,
				Description: t.Description,
				Account:     p.Account,
				Amount:      *p.Amount,
				Running:     running,
			})
		}
	}
	return entries
}
