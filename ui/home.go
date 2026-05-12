package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type HomeReadyMsg struct {
	Branch  string
	Changes int
	Err     error
}

type HomeModel struct {
	branch  string
	changes int
	ready   bool
}

func NewHomeModel() HomeModel {
	return HomeModel{}
}

func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	if msg, ok := msg.(HomeReadyMsg); ok {
		if msg.Err == nil {
			m.branch = msg.Branch
			m.changes = msg.Changes
		}
		m.ready = true
	}
	return m, nil
}

func (m HomeModel) View() string {
	var b strings.Builder
	b.WriteString("\n  " + headerStyle.Render("gitai"))

	if m.ready {
		b.WriteString("  " + branchStyle.Render("⎇ "+m.branch))
		if m.changes > 0 {
			b.WriteString("  " + warnStyle.Render(fmt.Sprintf("● %d changes", m.changes)))
		}
	}
	b.WriteString("\n\n")

	type menuItem struct{ key, label, desc string }
	items := []menuItem{
		{"c", "commit", "AI-powered commit grouping"},
		{"b", "branches", "switch or create branches"},
		{"l", "log", "view commit history"},
		{"p", "push", "push current branch to remote"},
		{"f", "pull", "pull from remote"},
		{"q", "quit", ""},
	}
	for _, item := range items {
		b.WriteString(fmt.Sprintf("  %s  %-10s  %s\n",
			keyStyle.Render(item.key),
			normalStyle.Render(item.label),
			dimStyle.Render(item.desc),
		))
	}
	return b.String()
}
