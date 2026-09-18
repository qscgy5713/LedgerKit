package main

import (
	"flag"
	"fmt"
)

func runAccounts(args []string) error {
	fs := flag.NewFlagSet("accounts", flag.ExitOnError)
	path := fs.String("journal", journalPath(), "journal file to read")
	fs.Parse(args)

	j, err := loadJournal(*path)
	if err != nil {
		return err
	}
	for _, a := range j.Accounts() {
		fmt.Println(a)
	}
	return nil
}
