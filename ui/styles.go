package ui

import "github.com/charmbracelet/lipgloss"

var (
	// text styles
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	fileStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).PaddingLeft(4)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	deletedStyle  = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("240"))
	countStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	doneStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	pendingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	failStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	branchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	keyStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	hashStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	dateStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	authorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("67"))
	aiStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Italic(true)
	remoteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statusOkStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	statusErrStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)
