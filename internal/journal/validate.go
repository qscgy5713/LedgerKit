package journal

import "fmt"

type ValidationError struct {
	Line int
	Msg  string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

// Validate checks structural invariants that Parse does not already enforce:
// every transaction must have at least 2 postings, and must balance to zero
// per commodity.
func (j *Journal) Validate() []error {
	var errs []error
	for _, t := range j.Transactions {
		if len(t.Postings) < 2 {
			errs = append(errs, ValidationError{Line: t.Line, Msg: fmt.Sprintf("transaction %q has fewer than 2 postings", t.Description)})
			continue
		}
		sums := map[string]Decimal{}
		for _, p := range t.Postings {
			if p.Amount == nil {
				continue
			}
			sums[p.Amount.Commodity] = sums[p.Amount.Commodity].Add(p.Amount.Value)
		}
		for c, v := range sums {
			if v.IsZero() {
				continue
			}
			label := c
			if label == "" {
				label = "(no commodity)"
			}
			errs = append(errs, ValidationError{
				Line: t.Line,
				Msg:  fmt.Sprintf("unbalanced transaction %q: %s off by %s", t.Description, label, v.String()),
			})
		}
	}
	return errs
}
