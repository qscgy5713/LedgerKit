package main

import (
	"flag"
	"fmt"
	"strings"

	"ledgerkit/internal/journal"
	"ledgerkit/internal/ledger"
	"ledgerkit/internal/render"
)

func runBalance(args []string) error {
	fs := flag.NewFlagSet("balance", flag.ExitOnError)
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
	fromT, err := journal.ParseOptionalDate(*from)
	if err != nil {
		return err
	}
	toT, err := journal.ParseOptionalDate(*to)
	if err != nil {
		return err
	}

	root := ledger.BuildTree(j.Transactions, ledger.Filter{AccountPattern: pattern, From: fromT, To: toT})

	var rows [][]string
	for _, r := range ledger.FlattenBalance(root) {
		for _, commodity := range r.Amounts.Commodities() {
			v := r.Amounts.Values[commodity]
			rows = append(rows, []string{
				strings.Repeat("  ", r.Depth) + r.Name,
				render.Amount(r.Amounts.Format(commodity), v.Sign()),
			})
		}
	}

	fmt.Print(render.Table([]string{"account", "balance"}, rows))
	for _, commodity := range root.Total.Commodities() {
		total := root.Total.Values[commodity]
		fmt.Printf("\n%s: %s\n", render.Heading("total"), render.Amount(root.Total.Format(commodity), total.Sign()))
	}
	return nil
}
