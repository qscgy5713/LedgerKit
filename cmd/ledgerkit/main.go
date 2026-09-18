// Command ledgerkit is a plain-text double-entry ledger CLI compatible with
// a subset of the hledger/ledger journal format.
package main

import (
	"fmt"
	"os"
)

var version = "dev"

type command struct {
	name string
	help string
	run  func(args []string) error
}

func commands() []command {
	return []command{
		{"init", "create a new journal file", runInit},
		{"add", "interactively add a transaction", runAdd},
		{"balance", "show account balances", runBalance},
		{"register", "show a chronological posting register", runRegister},
		{"import", "import transactions from a bank CSV export", runImport},
		{"accounts", "list all accounts used in the journal", runAccounts},
		{"check", "validate the journal", runCheck},
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: ledgerkit <command> [flags]")
	fmt.Fprintln(os.Stderr, "\ncommands:")
	for _, c := range commands() {
		fmt.Fprintf(os.Stderr, "  %-10s %s\n", c.name, c.help)
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-h", "--help", "help":
		usage()
		return
	case "-v", "--version", "version":
		fmt.Println("ledgerkit " + version)
		return
	}

	for _, c := range commands() {
		if c.name == os.Args[1] {
			if err := c.run(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "ledgerkit:", err)
				os.Exit(1)
			}
			return
		}
	}

	fmt.Fprintf(os.Stderr, "ledgerkit: unknown command %q\n\n", os.Args[1])
	usage()
	os.Exit(2)
}
