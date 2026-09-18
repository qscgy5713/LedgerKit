package journal

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Msg)
}

var (
	headerRe    = regexp.MustCompile(`^(\d{4})[-/](\d{2})[-/](\d{2})(?:\s+([*!]))?\s*(.*)$`)
	splitRe     = regexp.MustCompile(`\s{2,}|\t`)
	prefixAmtRe = regexp.MustCompile(`^([+-])?([^\d\s+-]+)\s*([+-]?[\d,]+(?:\.\d+)?)$`)
	suffixAmtRe = regexp.MustCompile(`^([+-]?[\d,]+(?:\.\d+)?)\s*([A-Za-z]+)$`)
	plainAmtRe  = regexp.MustCompile(`^[+-]?[\d,]+(?:\.\d+)?$`)
)

// Parse reads a hledger/ledger-style journal (MVP subset: no includes, tags,
// multi-commodity elision, or periodic transactions).
func Parse(r io.Reader) (*Journal, error) {
	scanner := bufio.NewScanner(r)
	var j Journal
	var cur *Transaction
	lineNo := 0

	flush := func() error {
		if cur == nil {
			return nil
		}
		if err := ResolveElided(cur); err != nil {
			return &ParseError{Line: cur.Line, Msg: err.Error()}
		}
		j.Transactions = append(j.Transactions, *cur)
		cur = nil
		return nil
	}

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}
		if strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue // standalone comment line, not attached to a transaction
		}

		indented := line[0] == ' ' || line[0] == '\t'
		if !indented {
			if err := flush(); err != nil {
				return nil, err
			}
			txn, err := parseHeader(line, lineNo)
			if err != nil {
				return nil, err
			}
			cur = txn
			continue
		}

		if cur == nil {
			return nil, &ParseError{Line: lineNo, Msg: "posting line without a preceding transaction header"}
		}
		p, err := parsePosting(line, lineNo)
		if err != nil {
			return nil, err
		}
		cur.Postings = append(cur.Postings, p)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return &j, nil
}

func splitComment(line string) (content, comment string) {
	idx := strings.Index(line, ";")
	if idx == -1 {
		return line, ""
	}
	return line[:idx], strings.TrimSpace(line[idx+1:])
}

func parseHeader(line string, lineNo int) (*Transaction, error) {
	content, comment := splitComment(line)
	content = strings.TrimSpace(content)

	m := headerRe.FindStringSubmatch(content)
	if m == nil {
		return nil, &ParseError{Line: lineNo, Msg: fmt.Sprintf("invalid transaction header: %q", content)}
	}
	date, err := time.Parse("2006-01-02", m[1]+"-"+m[2]+"-"+m[3])
	if err != nil {
		return nil, &ParseError{Line: lineNo, Msg: fmt.Sprintf("invalid date: %v", err)}
	}
	var status byte
	if m[4] != "" {
		status = m[4][0]
	}
	return &Transaction{
		Line:        lineNo,
		Date:        date,
		Status:      status,
		Description: strings.TrimSpace(m[5]),
		Comment:     comment,
	}, nil
}

func parsePosting(line string, lineNo int) (Posting, error) {
	content, comment := splitComment(line)
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return Posting{}, &ParseError{Line: lineNo, Msg: "empty posting line"}
	}

	var account, amountStr string
	if loc := splitRe.FindStringIndex(trimmed); loc != nil {
		account = strings.TrimSpace(trimmed[:loc[0]])
		amountStr = strings.TrimSpace(trimmed[loc[1]:])
	} else {
		account = trimmed
	}
	if account == "" {
		return Posting{}, &ParseError{Line: lineNo, Msg: "posting missing account name"}
	}
	if amountStr == "" {
		return Posting{Account: account, Comment: comment}, nil
	}
	amt, err := ParseAmount(amountStr)
	if err != nil {
		return Posting{}, &ParseError{Line: lineNo, Msg: err.Error()}
	}
	return Posting{Account: account, Amount: &amt, Comment: comment}, nil
}

// ParseAmount parses a single amount, e.g. "$120", "120 TWD", or "120".
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	if m := prefixAmtRe.FindStringSubmatch(s); m != nil {
		leadingSign, commodity, numStr := m[1], m[2], m[3]
		if leadingSign == "-" && numStr[0] != '+' && numStr[0] != '-' {
			numStr = "-" + numStr
		}
		val, err := ParseDecimal(numStr)
		if err != nil {
			return Amount{}, err
		}
		return Amount{Commodity: commodity, CommodityFirst: true, Value: val}, nil
	}
	if m := suffixAmtRe.FindStringSubmatch(s); m != nil {
		val, err := ParseDecimal(m[1])
		if err != nil {
			return Amount{}, err
		}
		return Amount{Commodity: m[2], CommodityFirst: false, Value: val}, nil
	}
	if plainAmtRe.MatchString(s) {
		val, err := ParseDecimal(s)
		if err != nil {
			return Amount{}, err
		}
		return Amount{Value: val}, nil
	}
	return Amount{}, fmt.Errorf("invalid amount: %q", s)
}

// ResolveElided fills in the at-most-one elided (nil Amount) posting in a
// transaction so it balances to zero. It is called automatically by Parse,
// and can be called again by callers (e.g. `add`, CSV import) that build
// transactions programmatically.
func ResolveElided(t *Transaction) error {
	elidedIdx := -1
	sums := map[string]Decimal{}
	for i, p := range t.Postings {
		if p.Amount == nil {
			if elidedIdx != -1 {
				return fmt.Errorf("transaction has more than one posting with no amount")
			}
			elidedIdx = i
			continue
		}
		sums[p.Amount.Commodity] = sums[p.Amount.Commodity].Add(p.Amount.Value)
	}
	if elidedIdx == -1 {
		return nil
	}
	if len(sums) == 0 {
		return fmt.Errorf("cannot infer amount: no other postings have an amount")
	}
	if len(sums) > 1 {
		return fmt.Errorf("cannot infer amount: transaction uses multiple commodities")
	}

	var commodity string
	var total Decimal
	for c, v := range sums {
		commodity, total = c, v
	}
	commodityFirst := false
	for _, p := range t.Postings {
		if p.Amount != nil && p.Amount.Commodity == commodity {
			commodityFirst = p.Amount.CommodityFirst
			break
		}
	}
	inferred := Amount{Commodity: commodity, CommodityFirst: commodityFirst, Value: total.Neg()}
	t.Postings[elidedIdx].Amount = &inferred
	return nil
}
