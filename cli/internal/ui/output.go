package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// JSON is toggled by the root --json persistent flag.
var JSON bool

// PrintJSON marshals v as indented JSON to stdout.
func PrintJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintTable renders a bubbles table to stdout.
func PrintTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println(WarningStyle.Render("No items found."))
		return
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	cols := make([]table.Column, len(headers))
	for i, h := range headers {
		cols[i] = table.Column{Title: h, Width: widths[i] + 2}
	}

	tblRows := make([]table.Row, len(rows))
	for i, r := range rows {
		tblRows[i] = table.Row(r)
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(tblRows),
		table.WithFocused(false),
		table.WithHeight(len(rows) + 1),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("6")).
		Foreground(lipgloss.Color("6")).
		Bold(true)
	s.Cell = s.Cell.Foreground(lipgloss.Color("15"))
	t.SetStyles(s)

	fmt.Println(t.View())
}

// DetailCard renders a titled key-value card.
func DetailCard(title string, pairs [][]string) {
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(KeyStyle.Render(p[0] + ":"))
		b.WriteString(" ")
		b.WriteString(ValueStyle.Render(p[1]))
		b.WriteString("\n")
	}
	fmt.Println(TitleStyle.Render(title))
	fmt.Print(CardStyle.Render(b.String()))
	fmt.Println()
}

// Die prints an error to stderr and exits 1.
func Die(err error) {
	fmt.Fprintln(os.Stderr, ErrorStyle.Render(err.Error()))
	os.Exit(1)
}

// Success prints a green success message.
func Success(msg string) {
	fmt.Println(SuccessStyle.Render(msg))
}
