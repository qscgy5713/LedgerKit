package main

import (
	"flag"
	"fmt"

	"ledgerkit/internal/importer"
	"ledgerkit/internal/journal"
)

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	path := fs.String("journal", journalPath(), "journal file to append imported transactions to")
	stateFile := fs.String("state", "", "dedupe state file (default: <csv path>.ledgerkit-import-state)")
	dateCol := fs.Int("date-col", 0, "0-based column index of the date")
	descCol := fs.Int("desc-col", 1, "0-based column index of the description")
	amountCol := fs.Int("amount-col", 2, "0-based column index of the amount")
	dateLayout := fs.String("date-layout", "2006-01-02", "Go reference layout for the date column")
	account := fs.String("account", "", "account this CSV statement belongs to, e.g. assets:bank:checking (required)")
	toAccount := fs.String("to", "expenses:unclassified", "counterparty account for imported rows")
	hasHeader := fs.Bool("header", false, "the CSV has a header row to skip")
	fs.Parse(args)

	if fs.NArg() < 1 {
		return fmt.Errorf("usage: ledgerkit import [flags] <csv-file>")
	}
	if *account == "" {
		return fmt.Errorf("-account is required")
	}
	csvPath := fs.Arg(0)
	if *stateFile == "" {
		*stateFile = csvPath + ".ledgerkit-import-state"
	}

	cfg := importer.Config{
		DateCol:    *dateCol,
		DescCol:    *descCol,
		AmountCol:  *amountCol,
		DateLayout: *dateLayout,
		Account:    *account,
		ToAccount:  *toAccount,
		HasHeader:  *hasHeader,
	}

	rows, err := importer.Import(csvPath, cfg, *stateFile)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		fmt.Println("no new rows to import")
		return nil
	}
	// Mark each row imported only after its transaction is durably in the
	// journal, so a failure partway through leaves the rest retryable
	// instead of silently lost or silently skipped on the next run.
	for i, r := range rows {
		if err := journal.AppendTransaction(*path, r.Transaction); err != nil {
			return fmt.Errorf("imported %d/%d rows before failing: %w", i, len(rows), err)
		}
		if err := importer.MarkImported(*stateFile, r); err != nil {
			return fmt.Errorf("imported %d/%d rows before failing to record import state: %w", i+1, len(rows), err)
		}
	}
	fmt.Printf("imported %d transaction(s) into %s\n", len(rows), *path)
	return nil
}
