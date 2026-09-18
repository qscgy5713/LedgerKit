package ledger

import (
	"strings"
	"testing"

	"ledgerkit/internal/journal"
)

func parseTestJournal(t *testing.T, src string) []journal.Transaction {
	t.Helper()
	j, err := journal.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return j.Transactions
}

const sampleJournal = `2026-09-01 * 買咖啡
    expenses:food:coffee    $120
    assets:cash

2026-09-02 * 買午餐
    expenses:food:lunch    $200
    assets:cash

2026-09-05 * 領薪水
    assets:cash    $50000
    income:salary
`

func TestBuildTreeTotals(t *testing.T) {
	txns := parseTestJournal(t, sampleJournal)
	root := BuildTree(txns, Filter{})

	assets := root.Children["assets"]
	if assets == nil {
		t.Fatalf("no assets node")
	}
	cash := assets.Children["cash"]
	if cash == nil || cash.Total.Values["$"].String() != "49680" {
		t.Fatalf("assets:cash total = %+v, want 49680", cash)
	}

	expenses := root.Children["expenses"]
	if expenses.Total.Values["$"].String() != "320" {
		t.Fatalf("expenses total = %s, want 320", expenses.Total.Values["$"].String())
	}
	food := expenses.Children["food"]
	if food.Total.Values["$"].String() != "320" {
		t.Fatalf("expenses:food total = %s, want 320", food.Total.Values["$"].String())
	}

	// whole-journal grand total across all top-level accounts must be zero
	grand := journal.Decimal{}
	for _, top := range root.SortedChildren() {
		grand = grand.Add(top.Total.Values["$"])
	}
	if !grand.IsZero() {
		t.Fatalf("grand total = %s, want 0", grand.String())
	}
}

func TestBuildTreeAccountFilter(t *testing.T) {
	txns := parseTestJournal(t, sampleJournal)
	root := BuildTree(txns, Filter{AccountPattern: "expenses"})
	if _, ok := root.Children["assets"]; ok {
		t.Fatalf("assets should be filtered out")
	}
	if root.Children["expenses"] == nil {
		t.Fatalf("expenses should be present")
	}
}

func TestBuildTreePreservesSuffixCommodityFormat(t *testing.T) {
	// A commodity written in suffix form ("1234.56 TWD") must render back
	// the same way in aggregated reports, not as a glued prefix "TWD1234.56".
	txns := parseTestJournal(t, `2026-09-01 領薪水
    assets:cash    1234.56 TWD
    income:salary
`)
	root := BuildTree(txns, Filter{})
	cash := root.Children["assets"].Children["cash"]
	if got := cash.Total.Format("TWD"); got != "1234.56 TWD" {
		t.Fatalf("got %q, want \"1234.56 TWD\"", got)
	}
}

func TestFlattenBalanceOrderAndDepth(t *testing.T) {
	txns := parseTestJournal(t, sampleJournal)
	root := BuildTree(txns, Filter{})
	rows := FlattenBalance(root)

	var got []string
	for _, r := range rows {
		got = append(got, strings.Repeat("  ", r.Depth)+r.Name+" ("+r.FullAccount+")")
	}
	want := []string{
		"assets (assets)",
		"  cash (assets:cash)",
		"expenses (expenses)",
		"  food (expenses:food)",
		"    coffee (expenses:food:coffee)",
		"    lunch (expenses:food:lunch)",
		"income (income)",
		"  salary (income:salary)",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}

	for _, r := range rows {
		if r.FullAccount == "assets:cash" && r.Amounts.Values["$"].String() != "49680" {
			t.Fatalf("assets:cash row amount = %s, want 49680", r.Amounts.Values["$"].String())
		}
	}
}

func TestRegisterRunningBalance(t *testing.T) {
	txns := parseTestJournal(t, sampleJournal)
	entries := Register(txns, Filter{AccountPattern: "assets:cash"})
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	last := entries[len(entries)-1]
	if last.Running.Values["$"].String() != "49680" {
		t.Fatalf("final running balance = %s, want 49680", last.Running.Values["$"].String())
	}
	if !entries[0].Date.Before(entries[1].Date) {
		t.Fatalf("entries not sorted by date")
	}
}
