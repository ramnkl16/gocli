// Package ui contains small helpers for pretty terminal output.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

var (
	// Title is used for section headers like ">> Pull Requests".
	Title = color.New(color.FgCyan, color.Bold).SprintFunc()
	// Dim is used for secondary text.
	Dim = color.New(color.FgHiBlack).SprintFunc()
	// OK is used for success messages.
	OK = color.New(color.FgGreen, color.Bold).SprintFunc()
	// Warn is used for warnings.
	Warn = color.New(color.FgYellow, color.Bold).SprintFunc()
	// Err is used for errors.
	Err = color.New(color.FgRed, color.Bold).SprintFunc()
)

// Section prints a header bar to stdout.
func Section(s string) {
	fmt.Println()
	fmt.Println(Title("» " + s))
	fmt.Println(Dim(strings.Repeat("─", len(s)+2)))
}

// Table writes a small ASCII table to w. headers should match each row's len.
func Table(w io.Writer, headers []string, rows [][]string) {
	if w == nil {
		w = os.Stdout
	}
	t := tablewriter.NewTable(
		w,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Separators: tw.Separators{BetweenColumns: tw.On},
			},
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Alignment:  tw.CellAlignment{Global: tw.AlignLeft},
				Formatting: tw.CellFormatting{AutoFormat: tw.On},
			},
			Row: tw.CellConfig{
				Alignment: tw.CellAlignment{Global: tw.AlignLeft},
			},
		}),
	)
	t.Header(headers)
	t.Bulk(rows)
	_ = t.Render()
}

// Truncate shortens s to n runes, appending "…" if cut.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// Hyperlink wraps label with an OSC 8 hyperlink to url when stdout is a TTY
// and GOCLI_NO_HYPERLINK is not set. In unsupported or non-interactive
// contexts it returns label unchanged (no escape sequences).
//
// In Windows Terminal / VS Code / iTerm, Ctrl+Click (or Cmd+Click) the
// visible text opens the URL in the default browser.
func Hyperlink(targetURL, label string) string {
	if os.Getenv("GOCLI_NO_HYPERLINK") != "" {
		return label
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return label
	}
	if targetURL == "" || label == "" {
		return label
	}
	const esc = "\x1b"
	// OSC 8: ESC ] 8 ; ; url ST  text  ESC ] 8 ; ; ST
	return esc + "]8;;" + targetURL + esc + "\\" + label + esc + "]8;;" + esc + "\\"
}
