package journal

import "sort"

// Amounts accumulates per-commodity totals while remembering each
// commodity's first-seen display convention (prefix "$120" vs. suffix
// "120 TWD"), so aggregated reports (balance, register) can render the same
// way the source journal did instead of always falling back to one style.
type Amounts struct {
	Values map[string]Decimal
	prefix map[string]bool
}

func NewAmounts() Amounts {
	return Amounts{Values: map[string]Decimal{}, prefix: map[string]bool{}}
}

// Add returns a new Amounts with amt folded in; the receiver is left
// unmodified so callers can keep cheap point-in-time snapshots (used by the
// register report's running balance column).
func (a Amounts) Add(amt Amount) Amounts {
	out := Amounts{
		Values: make(map[string]Decimal, len(a.Values)+1),
		prefix: make(map[string]bool, len(a.prefix)+1),
	}
	for k, v := range a.Values {
		out.Values[k] = v
	}
	for k, v := range a.prefix {
		out.prefix[k] = v
	}
	out.Values[amt.Commodity] = out.Values[amt.Commodity].Add(amt.Value)
	if _, ok := out.prefix[amt.Commodity]; !ok {
		out.prefix[amt.Commodity] = amt.CommodityFirst
	}
	return out
}

// Merge folds another Amounts' totals into a new Amounts. Where both sides
// already know a commodity's display convention, the receiver's wins.
func (a Amounts) Merge(other Amounts) Amounts {
	out := Amounts{
		Values: make(map[string]Decimal, len(a.Values)+len(other.Values)),
		prefix: make(map[string]bool, len(a.prefix)+len(other.prefix)),
	}
	for k, v := range a.Values {
		out.Values[k] = v
	}
	for k, v := range a.prefix {
		out.prefix[k] = v
	}
	for k, v := range other.Values {
		out.Values[k] = out.Values[k].Add(v)
		if _, ok := out.prefix[k]; !ok {
			out.prefix[k] = other.prefix[k]
		}
	}
	return out
}

// Commodities returns the commodities present, sorted, so rendering order
// is stable across runs (Go map iteration order is randomized).
func (a Amounts) Commodities() []string {
	out := make([]string, 0, len(a.Values))
	for c := range a.Values {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// Format renders the total for one commodity using its remembered
// prefix/suffix convention.
func (a Amounts) Format(commodity string) string {
	return Amount{Commodity: commodity, CommodityFirst: a.prefix[commodity], Value: a.Values[commodity]}.String()
}
