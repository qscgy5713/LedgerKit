package journal

import (
	"os"
	"strings"
)

// FormatTransaction renders a transaction back into journal syntax.
func FormatTransaction(t Transaction) string {
	var b strings.Builder
	b.WriteString(t.Date.Format("2006-01-02"))
	if t.Status != 0 {
		b.WriteByte(' ')
		b.WriteByte(t.Status)
	}
	if t.Description != "" {
		b.WriteByte(' ')
		b.WriteString(t.Description)
	}
	if t.Comment != "" {
		b.WriteString("  ; ")
		b.WriteString(t.Comment)
	}
	b.WriteByte('\n')
	for _, p := range t.Postings {
		b.WriteString("    ")
		b.WriteString(p.Account)
		if p.Amount != nil {
			b.WriteString("    ")
			b.WriteString(p.Amount.String())
		}
		if p.Comment != "" {
			b.WriteString("  ; ")
			b.WriteString(p.Comment)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// AppendTransaction appends a transaction to the journal file at path,
// creating it if necessary and separating it from prior content with a
// blank line.
func AppendTransaction(path string, t Transaction) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if info, err := f.Stat(); err == nil && info.Size() > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(FormatTransaction(t))
	return err
}
