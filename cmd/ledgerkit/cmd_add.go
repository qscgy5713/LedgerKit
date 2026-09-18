package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"ledgerkit/internal/journal"
)

func runAdd(args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	path := fs.String("journal", journalPath(), "journal file to append to")
	fs.Parse(args)

	in := bufio.NewReader(os.Stdin)
	txn, err := promptTransaction(in, os.Stdout)
	if err != nil {
		return err
	}

	if err := journal.AppendTransaction(*path, txn); err != nil {
		return err
	}
	fmt.Printf("added to %s:\n\n%s", *path, journal.FormatTransaction(txn))
	return nil
}

func promptTransaction(in *bufio.Reader, out *os.File) (journal.Transaction, error) {
	dateStr, err := prompt(in, out, fmt.Sprintf("date [%s]: ", time.Now().Format("2006-01-02")))
	if err != nil {
		return journal.Transaction{}, err
	}
	date := time.Now()
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return journal.Transaction{}, fmt.Errorf("invalid date: %w", err)
		}
	}

	desc, err := prompt(in, out, "description: ")
	if err != nil {
		return journal.Transaction{}, err
	}
	if desc == "" {
		return journal.Transaction{}, fmt.Errorf("description is required")
	}

	txn := journal.Transaction{Date: date, Status: '*', Description: desc}
	fmt.Fprintln(out, "enter postings (blank account to finish; leave amount blank to auto-balance the last posting):")
	for {
		account, err := prompt(in, out, fmt.Sprintf("  account %d: ", len(txn.Postings)+1))
		if err != nil {
			return journal.Transaction{}, err
		}
		if account == "" {
			break
		}
		amountStr, err := prompt(in, out, "  amount (blank to infer): ")
		if err != nil {
			return journal.Transaction{}, err
		}
		p := journal.Posting{Account: account}
		if amountStr != "" {
			amt, err := journal.ParseAmount(amountStr)
			if err != nil {
				return journal.Transaction{}, err
			}
			p.Amount = &amt
		}
		txn.Postings = append(txn.Postings, p)
	}

	if len(txn.Postings) < 2 {
		return journal.Transaction{}, fmt.Errorf("need at least 2 postings")
	}
	if err := journal.ResolveElided(&txn); err != nil {
		return journal.Transaction{}, err
	}
	return txn, nil
}

func prompt(in *bufio.Reader, out *os.File, label string) (string, error) {
	fmt.Fprint(out, label)
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
