package journal

import (
	"fmt"
	"strconv"
	"strings"
)

// Decimal is a fixed-point decimal number represented as an integer
// (Unscaled) with an implied number of fractional digits (Scale).
// It avoids float64 rounding error for money math.
type Decimal struct {
	Unscaled int64
	Scale    uint8
}

func ParseDecimal(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, fmt.Errorf("empty amount")
	}

	neg := false
	switch {
	case strings.HasPrefix(s, "-"):
		neg = true
		s = s[1:]
	case strings.HasPrefix(s, "+"):
		s = s[1:]
	}

	s = strings.ReplaceAll(s, ",", "")

	intPart, fracPart, hasFrac := strings.Cut(s, ".")
	if intPart == "" {
		intPart = "0"
	}
	if hasFrac && fracPart == "" {
		return Decimal{}, fmt.Errorf("invalid amount %q", s)
	}
	for _, r := range intPart {
		if r < '0' || r > '9' {
			return Decimal{}, fmt.Errorf("invalid amount %q", s)
		}
	}
	for _, r := range fracPart {
		if r < '0' || r > '9' {
			return Decimal{}, fmt.Errorf("invalid amount %q", s)
		}
	}

	scale := uint8(len(fracPart))
	n, err := strconv.ParseInt(intPart+fracPart, 10, 64)
	if err != nil {
		return Decimal{}, fmt.Errorf("amount out of range: %q", s)
	}
	if neg {
		n = -n
	}
	return Decimal{Unscaled: n, Scale: scale}, nil
}

func pow10(n uint8) int64 {
	p := int64(1)
	for i := uint8(0); i < n; i++ {
		p *= 10
	}
	return p
}

func (d Decimal) rescale(scale uint8) Decimal {
	if d.Scale == scale {
		return d
	}
	if scale > d.Scale {
		return Decimal{Unscaled: d.Unscaled * pow10(scale-d.Scale), Scale: scale}
	}
	return Decimal{Unscaled: d.Unscaled / pow10(d.Scale-scale), Scale: scale}
}

func maxScale(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

func (d Decimal) Add(o Decimal) Decimal {
	s := maxScale(d.Scale, o.Scale)
	d2, o2 := d.rescale(s), o.rescale(s)
	return Decimal{Unscaled: d2.Unscaled + o2.Unscaled, Scale: s}
}

func (d Decimal) Neg() Decimal {
	return Decimal{Unscaled: -d.Unscaled, Scale: d.Scale}
}

func (d Decimal) IsZero() bool {
	return d.Unscaled == 0
}

func (d Decimal) Sign() int {
	switch {
	case d.Unscaled > 0:
		return 1
	case d.Unscaled < 0:
		return -1
	default:
		return 0
	}
}

func (d Decimal) String() string {
	neg := d.Unscaled < 0
	u := d.Unscaled
	if neg {
		u = -u
	}
	s := strconv.FormatInt(u, 10)
	if d.Scale == 0 {
		if neg {
			return "-" + s
		}
		return s
	}
	for len(s) <= int(d.Scale) {
		s = "0" + s
	}
	cut := len(s) - int(d.Scale)
	out := s[:cut] + "." + s[cut:]
	if neg {
		out = "-" + out
	}
	return out
}
