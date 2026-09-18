package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bank.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportBasic(t *testing.T) {
	csvPath := writeTempCSV(t, "date,desc,amount\n2026-09-18,咖啡,-120\n2026-09-19,薪水,50000\n")
	stateFile := filepath.Join(t.TempDir(), "state")

	cfg := Config{DateCol: 0, DescCol: 1, AmountCol: 2, HasHeader: true, Account: "assets:cash"}
	rows, err := Import(csvPath, cfg, stateFile)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	txn := rows[0].Transaction
	if txn.Postings[1].Account != "expenses:unclassified" {
		t.Fatalf("got counterparty %q", txn.Postings[1].Account)
	}
	if txn.Postings[1].Amount.Value.String() != "120" {
		t.Fatalf("elided counterparty amount = %s, want 120", txn.Postings[1].Amount.Value.String())
	}
}

func TestImportDedupesOnRerunAfterMarkImported(t *testing.T) {
	csvPath := writeTempCSV(t, "2026-09-18,咖啡,-120\n2026-09-19,薪水,50000\n")
	stateFile := filepath.Join(t.TempDir(), "state")
	cfg := Config{DateCol: 0, DescCol: 1, AmountCol: 2, Account: "assets:cash"}

	first, err := Import(csvPath, cfg, stateFile)
	if err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("first import: got %d rows, want 2", len(first))
	}
	if err := MarkImported(stateFile, first...); err != nil {
		t.Fatalf("MarkImported() error = %v", err)
	}

	second, err := Import(csvPath, cfg, stateFile)
	if err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second import: got %d rows, want 0 (all should be deduped)", len(second))
	}
}

func TestImportWithoutMarkImportedIsNotDeduped(t *testing.T) {
	// This is the behavior the persist-before-append ordering bug depended
	// on: Import() must not, by itself, mark anything as seen. Only
	// MarkImported does, so a caller that fails before calling it can
	// safely retry without losing rows.
	csvPath := writeTempCSV(t, "2026-09-18,咖啡,-120\n")
	stateFile := filepath.Join(t.TempDir(), "state")
	cfg := Config{DateCol: 0, DescCol: 1, AmountCol: 2, Account: "assets:cash"}

	if _, err := Import(csvPath, cfg, stateFile); err != nil {
		t.Fatalf("first Import() error = %v", err)
	}
	again, err := Import(csvPath, cfg, stateFile)
	if err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if len(again) != 1 {
		t.Fatalf("got %d rows, want 1 (nothing should be marked seen without MarkImported)", len(again))
	}
}

func TestImportNewRowsAfterPartialSeen(t *testing.T) {
	csvPath := writeTempCSV(t, "2026-09-18,咖啡,-120\n")
	stateFile := filepath.Join(t.TempDir(), "state")
	cfg := Config{DateCol: 0, DescCol: 1, AmountCol: 2, Account: "assets:cash"}

	first, err := Import(csvPath, cfg, stateFile)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if err := MarkImported(stateFile, first...); err != nil {
		t.Fatalf("MarkImported() error = %v", err)
	}

	// simulate the bank statement growing with one new row appended
	if err := os.WriteFile(csvPath, []byte("2026-09-18,咖啡,-120\n2026-09-20,午餐,-80\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := Import(csvPath, cfg, stateFile)
	if err != nil {
		t.Fatalf("second Import() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Transaction.Description != "午餐" {
		t.Fatalf("got %+v, want exactly the new 午餐 row", rows)
	}
}

func TestImportRejectsNegativeColumnIndex(t *testing.T) {
	csvPath := writeTempCSV(t, "2026-09-18,咖啡,-120\n")
	stateFile := filepath.Join(t.TempDir(), "state")
	cfg := Config{DateCol: -1, DescCol: 1, AmountCol: 2, Account: "assets:cash"}

	if _, err := Import(csvPath, cfg, stateFile); err == nil {
		t.Fatalf("expected an error for a negative column index, got nil")
	}
}
