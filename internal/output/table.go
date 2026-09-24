package output

import (
	"os"
	"strings"
	"unicode"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// Table is a tablewriter.Table whose rows are cleaned of control characters.
type Table struct{ *tablewriter.Table }

// NewTable creates a pre-styled table ready to Append rows and Render.
//
// Style: no outer border, left-aligned upper-cased headers, a dashed line
// under the header, two-space gutters between columns, and no line wrapping.
func NewTable(headers []string) *Table {
	symbols := tw.NewSymbolCustom("avochato").
		WithRow("-").
		WithColumn("  ").
		WithCenter(" ")

	t := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off},
			Symbols: symbols,
			Settings: tw.Settings{
				Separators: tw.Separators{
					ShowHeader:     tw.On,
					ShowFooter:     tw.Off,
					BetweenRows:    tw.Off,
					BetweenColumns: tw.On,
				},
				Lines: tw.Lines{
					ShowTop:        tw.Off,
					ShowBottom:     tw.Off,
					ShowHeaderLine: tw.On,
					ShowFooterLine: tw.Off,
				},
			},
		}),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAutoFormat(tw.On),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
	)
	t.Header(headers)
	return &Table{t}
}

// Append adds a row with every cell passed through Clean.
func (t *Table) Append(row []string) {
	for i := range row {
		row[i] = Clean(row[i])
	}
	// The underlying Append only fails on unsupported row types; a []string
	// is always accepted, so the error is safe to drop here.
	_ = t.Table.Append(row)
}

// Clean strips terminal control characters from third-party text before it is printed.
func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return ' '
		case unicode.IsPrint(r):
			return r
		}
		return -1
	}, s)
}

// BoolStr converts a bool to a human-readable "yes" / "no".
func BoolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
