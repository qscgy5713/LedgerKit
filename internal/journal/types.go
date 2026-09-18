package journal

import (
	"sort"
	"time"
)

// Amount is a quantity of some commodity (currency), e.g. "$120" or "120 TWD".
type Amount struct {
	Commodity      string
	CommodityFirst bool // true: "$120", false: "120 TWD", irrelevant when Commodity == ""
	Value          Decimal
}

func (a Amount) Neg() Amount {
	return Amount{Commodity: a.Commodity, CommodityFirst: a.CommodityFirst, Value: a.Value.Neg()}
}

func (a Amount) String() string {
	if a.Commodity == "" {
		return a.Value.String()
	}
	if a.CommodityFirst {
		return a.Commodity + a.Value.String()
	}
	return a.Value.String() + " " + a.Commodity
}

// Posting is one leg of a transaction. Amount is nil when the amount was
// elided in the source (to be inferred so the transaction balances to zero).
type Posting struct {
	Account string
	Amount  *Amount
	Comment string
}

type Transaction struct {
	Line        int // 1-based source line number of the header, for error messages
	Date        time.Time
	Status      byte // 0, '*' (cleared) or '!' (pending)
	Description string
	Comment     string
	Postings    []Posting
}

type Journal struct {
	Transactions []Transaction
}

// Accounts returns the sorted set of distinct account names used anywhere in the journal.
func (j *Journal) Accounts() []string {
	set := map[string]bool{}
	for _, t := range j.Transactions {
		for _, p := range t.Postings {
			set[p.Account] = true
		}
	}
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
