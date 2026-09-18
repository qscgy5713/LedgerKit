package importer

import (
	"bufio"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"ledgerkit/internal/journal"
)

// Config describes how to map a generic bank CSV export onto a transaction.
// v1 is deliberately generic (column indices, not a per-bank format) —
// dedicated per-bank parsers can be added later once a real sample exists.
type Config struct {
	DateCol    int
	DescCol    int
	AmountCol  int
	DateLayout string // Go reference layout, default "2006-01-02"
	Account    string // account the CSV belongs to, e.g. "assets:bank:checking"
	ToAccount  string // counterparty account, default "expenses:unclassified"
	HasHeader  bool
}

func (c Config) withDefaults() Config {
	if c.DateLayout == "" {
		c.DateLayout = "2006-01-02"
	}
	if c.ToAccount == "" {
		c.ToAccount = "expenses:unclassified"
	}
	return c
}

func (c Config) validate() error {
	for name, col := range map[string]int{"date-col": c.DateCol, "desc-col": c.DescCol, "amount-col": c.AmountCol} {
		if col < 0 {
			return fmt.Errorf("%s must be >= 0, got %d", name, col)
		}
	}
	return nil
}

// Row pairs a parsed transaction with the hash of its source CSV row, so a
// caller can mark it imported (via MarkImported) only after successfully
// persisting it elsewhere.
type Row struct {
	Transaction journal.Transaction
	hash        string
}

func loadSeen(stateFile string) (map[string]bool, error) {
	seen := map[string]bool{}
	f, err := os.Open(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return seen, nil
		}
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			seen[line] = true
		}
	}
	return seen, sc.Err()
}

func hashRow(row []string) string {
	sum := sha256.Sum256([]byte(strings.Join(row, "\x1f")))
	return hex.EncodeToString(sum[:])
}

// Import reads csvPath and returns the rows not already recorded in
// stateFile, as transactions. It does NOT record anything in stateFile —
// call MarkImported for each row only after it has been durably persisted
// (e.g. appended to the journal), so a failure partway through a batch
// leaves the not-yet-persisted rows eligible to be retried instead of
// silently skipped forever.
func Import(csvPath string, cfg Config, stateFile string) ([]Row, error) {
	cfg = cfg.withDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	csvRows, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse csv: %w", err)
	}
	if cfg.HasHeader && len(csvRows) > 0 {
		csvRows = csvRows[1:]
	}

	seen, err := loadSeen(stateFile)
	if err != nil {
		return nil, fmt.Errorf("read import state: %w", err)
	}

	maxCol := cfg.DateCol
	if cfg.DescCol > maxCol {
		maxCol = cfg.DescCol
	}
	if cfg.AmountCol > maxCol {
		maxCol = cfg.AmountCol
	}

	var rows []Row
	for i, csvRow := range csvRows {
		if len(csvRow) <= maxCol {
			return nil, fmt.Errorf("csv row %d has only %d columns, need at least %d", i+1, len(csvRow), maxCol+1)
		}
		h := hashRow(csvRow)
		if seen[h] {
			continue
		}

		date, err := time.Parse(cfg.DateLayout, strings.TrimSpace(csvRow[cfg.DateCol]))
		if err != nil {
			return nil, fmt.Errorf("csv row %d: invalid date %q: %w", i+1, csvRow[cfg.DateCol], err)
		}
		val, err := journal.ParseDecimal(strings.TrimSpace(csvRow[cfg.AmountCol]))
		if err != nil {
			return nil, fmt.Errorf("csv row %d: invalid amount %q: %w", i+1, csvRow[cfg.AmountCol], err)
		}

		amt := journal.Amount{Value: val}
		txn := journal.Transaction{
			Date:        date,
			Status:      '*',
			Description: strings.TrimSpace(csvRow[cfg.DescCol]),
			Postings: []journal.Posting{
				{Account: cfg.Account, Amount: &amt},
				{Account: cfg.ToAccount},
			},
		}
		if err := journal.ResolveElided(&txn); err != nil {
			return nil, fmt.Errorf("csv row %d: %w", i+1, err)
		}

		rows = append(rows, Row{Transaction: txn, hash: h})
	}

	return rows, nil
}

// MarkImported records rows as imported in stateFile so a future Import of
// the same CSV skips them. Call it once per row, right after that row's
// transaction has been durably persisted.
func MarkImported(stateFile string, rows ...Row) error {
	if len(rows) == 0 {
		return nil
	}
	f, err := os.OpenFile(stateFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, r := range rows {
		if _, err := fmt.Fprintln(f, r.hash); err != nil {
			return err
		}
	}
	return nil
}
