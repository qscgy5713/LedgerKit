package journal

import (
	"fmt"
	"os"
	"time"
)

// LoadFile opens and parses the journal at path. Callers that want a
// friendlier message for a missing file should check os.IsNotExist(err)
// themselves — this only wraps the read+parse, not presentation.
func LoadFile(path string) (*Journal, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// ParseOptionalDate parses a "YYYY-MM-DD" date filter value. An empty
// string is not an error — it returns (nil, nil), meaning "no filter".
// Shared by the CLI's -from/-to flags and the HTTP API's ?from=&to= query
// params so both accept exactly the same date syntax.
func ParseOptionalDate(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q, want YYYY-MM-DD", s)
	}
	return &t, nil
}
