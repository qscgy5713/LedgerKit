package main

import (
	"flag"
	"fmt"

	"ledgerkit/internal/ledger"
	"ledgerkit/internal/render"
)

func runRegister(args []string) error {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	path := fs.String("journal", journalPath(), "journal file to read")
	from := fs.String("from", "", "only include postings on/after this date (YYYY-MM-DD)")
	to := fs.String("to", "", "only include postings on/before this date (YYYY-MM-DD)")
	fs.Parse(args)

	pattern := ""
	if fs.NArg() > 0 {
		pattern = fs.Arg(0)
	}

	j, err := loadJournal(*path)
	if err != nil {
		return err
	}
	fromT, err := parseDateFlag(*from)
	if err != nil {
		return err
	}
	toT, err := parseDateFlag(*to)
	if err != nil {
		return err
	}

	entries := ledger.Register(j.Transactions, ledger.Filter{AccountPattern: pattern, From: fromT, To: toT})

	var rows [][]string
	for _, e := range entries {
		var running string
		for _, commodity := range e.Running.Commodities() {
			if running != "" {
				running += ", "
			}
			running += render.Amount(e.Running.Format(commodity), e.Running.Values[commodity].Sign())
		}
		rows = append(rows, []string{
			e.Date.Format("2006-01-02"),
			e.Description,
			e.Account,
			render.Amount(e.Amount.String(), e.Amount.Value.Sign()),
			running,
		})
	}
	fmt.Print(render.Table([]string{"date", "description", "account", "amount", "balance"}, rows))
	return nil
}
