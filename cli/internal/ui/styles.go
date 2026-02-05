package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle   = lipgloss.NewStyle().Bold(true).Underline(true).MarginBottom(1)
	KeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true).Width(18)
	ValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	CardStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("6")).Padding(0, 1).MarginTop(1)
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	LogsStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color("4")).Padding(0, 1).MarginTop(1)
)

// StatusBadge returns a lipgloss-coloured string for a known status value.
func StatusBadge(status string) string {
	switch status {
	case "success", "ok":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(status)
	case "running", "pending":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Render(status)
	case "failed":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Render(status)
	default:
		return status
	}
}
