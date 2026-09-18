package main

import (
	"flag"
	"fmt"
	"os"
)

const exampleJournal = `; ledgerkit journal
; 2026-09-18 * 買咖啡
;     expenses:food:coffee    $120
;     assets:cash

2026-01-01 * 期初餘額
    assets:cash    $1000
    equity:opening-balance
`

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	force := fs.Bool("force", false, "overwrite the journal file if it already exists")
	fs.Parse(args)

	path := journalPath()
	if len(fs.Args()) > 0 {
		path = fs.Args()[0]
	}

	if _, err := os.Stat(path); err == nil && !*force {
		return fmt.Errorf("%q already exists (use -force to overwrite)", path)
	}

	if err := os.WriteFile(path, []byte(exampleJournal), 0o644); err != nil {
		return err
	}
	fmt.Printf("created %s\n", path)
	return nil
}
