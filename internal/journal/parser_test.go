package journal

import (
	"strings"
	"testing"
	"time"
)

func mustAmount(t *testing.T, p Posting) Amount {
	t.Helper()
	if p.Amount == nil {
		t.Fatalf("posting %q has nil amount", p.Account)
	}
	return *p.Amount
}

func TestParseBasicTransaction(t *testing.T) {
	src := `2026-09-18 * 買咖啡
    expenses:food:coffee    $120
    assets:cash
`
	j, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(j.Transactions) != 1 {
		t.Fatalf("got %d transactions, want 1", len(j.Transactions))
	}
	tx := j.Transactions[0]
	if tx.Status != '*' || tx.Description != "買咖啡" {
		t.Fatalf("got status=%q desc=%q", tx.Status, tx.Description)
	}
	if len(tx.Postings) != 2 {
		t.Fatalf("got %d postings, want 2", len(tx.Postings))
	}
	a := mustAmount(t, tx.Postings[0])
	if a.Commodity != "$" || !a.CommodityFirst || a.Value.String() != "120" {
		t.Fatalf("got amount %+v", a)
	}
	b := mustAmount(t, tx.Postings[1])
	if b.Value.String() != "-120" {
		t.Fatalf("elided posting resolved to %+v, want -120", b)
	}
}

func TestParseLeadingSignBeforeCommodity(t *testing.T) {
	src := `2026-09-18 退款
    assets:cash    -$120
    income:refunds
`
	j, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	a := mustAmount(t, j.Transactions[0].Postings[0])
	if a.Value.String() != "-120" {
		t.Fatalf("got %+v, want -120", a)
	}
	b := mustAmount(t, j.Transactions[0].Postings[1])
	if b.Value.String() != "120" {
		t.Fatalf("elided posting resolved to %+v, want 120", b)
	}
}

func TestParseSuffixCommodityAndDecimals(t *testing.T) {
	src := `2026-09-18 領薪水
    assets:cash    1234.56 TWD
    income:salary    -1234.56 TWD
`
	j, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	a := mustAmount(t, j.Transactions[0].Postings[0])
	if a.Commodity != "TWD" || a.CommodityFirst || a.Value.String() != "1234.56" {
		t.Fatalf("got %+v", a)
	}
}

func TestParseMultipleElidedFails(t *testing.T) {
	src := `2026-09-18 bad
    assets:cash
    expenses:food
`
	if _, err := Parse(strings.NewReader(src)); err == nil {
		t.Fatalf("expected error for two elided postings, got nil")
	}
}

func TestParseCommentsAndBlankLines(t *testing.T) {
	src := `; this is a top-level comment

2026-09-18 * 買咖啡  ; txn comment
    expenses:food:coffee    $120  ; posting comment
    assets:cash

2026-09-19 買午餐
    expenses:food:lunch    $80
    assets:cash
`
	j, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(j.Transactions) != 2 {
		t.Fatalf("got %d transactions, want 2", len(j.Transactions))
	}
	if j.Transactions[0].Comment != "txn comment" {
		t.Fatalf("got comment %q", j.Transactions[0].Comment)
	}
	if j.Transactions[0].Postings[0].Comment != "posting comment" {
		t.Fatalf("got posting comment %q", j.Transactions[0].Postings[0].Comment)
	}
}

func TestParseInvalidHeader(t *testing.T) {
	src := "not-a-date some text\n    assets:cash    $1\n"
	_, err := Parse(strings.NewReader(src))
	if err == nil {
		t.Fatalf("expected error for invalid header")
	}
	var perr *ParseError
	if !asParseError(err, &perr) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if perr.Line != 1 {
		t.Fatalf("got line %d, want 1", perr.Line)
	}
}

func asParseError(err error, target **ParseError) bool {
	if pe, ok := err.(*ParseError); ok {
		*target = pe
		return true
	}
	return false
}

func TestPostingWithoutHeader(t *testing.T) {
	src := "    assets:cash    $1\n"
	if _, err := Parse(strings.NewReader(src)); err == nil {
		t.Fatalf("expected error for posting without header")
	}
}

func TestValidateDetectsUnbalanced(t *testing.T) {
	j := &Journal{Transactions: []Transaction{
		{
			Line:        1,
			Date:        mustParseDate(t, "2026-09-18"),
			Description: "unbalanced",
			Postings: []Posting{
				{Account: "assets:cash", Amount: &Amount{Value: mustDecimal(t, "100")}},
				{Account: "expenses:food", Amount: &Amount{Value: mustDecimal(t, "-90")}},
			},
		},
	}}
	errs := j.Validate()
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return tm
}

func mustDecimal(t *testing.T, s string) Decimal {
	t.Helper()
	d, err := ParseDecimal(s)
	if err != nil {
		t.Fatalf("bad decimal %q: %v", s, err)
	}
	return d
}
