package main

import (
	"flag"
	"fmt"

	"ledgerkit/internal/render"
)

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	path := fs.String("journal", journalPath(), "journal file to check")
	fs.Parse(args)

	j, err := loadJournal(*path)
	if err != nil {
		return err
	}

	errs := j.Validate()
	if len(errs) == 0 {
		fmt.Printf("%s: %d transactions, all balanced\n", *path, len(j.Transactions))
		return nil
	}
	for _, e := range errs {
		fmt.Println(render.Amount(e.Error(), -1))
	}
	return fmt.Errorf("%d problem(s) found", len(errs))
}
