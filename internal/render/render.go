// Package render formats CLI output: aligned tables and sign-colored amounts.
package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

var (
	positiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	negativeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	zeroStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headingStyle  = lipgloss.NewStyle().Bold(true)
)

// Amount colors a formatted amount string by its sign: green (positive),
// red (negative), or dim gray (zero).
func Amount(s string, sign int) string {
	switch {
	case sign > 0:
		return positiveStyle.Render(s)
	case sign < 0:
		return negativeStyle.Render(s)
	default:
		return zeroStyle.Render(s)
	}
}

func Heading(s string) string {
	return headingStyle.Render(s)
}

// Table renders headers and rows as an aligned ASCII table. Cell content is
// not space-trimmed, so callers can use leading spaces to show hierarchy
// (see cmd_balance.go's indented account tree).
func Table(headers []string, rows [][]string) string {
	var b strings.Builder
	t := tablewriter.NewTable(&b, tablewriter.WithTrimSpace(tw.Off))
	t.Header(headers)
	if len(rows) > 0 {
		_ = t.Bulk(rows)
	}
	_ = t.Render()
	return b.String()
}
