package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/finn/gitai/ai"
)

var (
	selectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	normalStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	fileStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).PaddingLeft(4)
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	deletedStyle   = lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("240"))
	headerStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	countStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

type ReviewModel struct {
	commits  []ai.CommitGroup
	deleted  map[int]bool
	cursor   int
	editing  bool
	input    textinput.Model
}

func NewReviewModel(commits []ai.CommitGroup) ReviewModel {
	ti := textinput.New()
	ti.CharLimit = 200
	return ReviewModel{
		commits: commits,
		deleted: make(map[int]bool),
		input:   ti,
	}
}

func (m ReviewModel) Update(msg tea.Msg) (ReviewModel, tea.Cmd) {
	if m.editing {
		return m.updateEditing(msg)
	}
	return m.updateNavigating(msg)
}

func (m ReviewModel) updateEditing(msg tea.Msg) (ReviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.commits[m.cursor].Message = m.input.Value()
			m.editing = false
			m.input.Blur()
			return m, nil
		case tea.KeyEsc:
			m.editing = false
			m.input.Blur()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m ReviewModel) updateNavigating(msg tea.Msg) (ReviewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.commits)-1 {
				m.cursor++
			}
		case "e":
			if !m.deleted[m.cursor] {
				m.input.SetValue(m.commits[m.cursor].Message)
				m.input.Focus()
				m.editing = true
			}
		case "d":
			m.deleted[m.cursor] = !m.deleted[m.cursor]
		}
	}
	return m, nil
}

func (m ReviewModel) IsEditing() bool {
	return m.editing
}

func (m ReviewModel) ActiveCommits() []ai.CommitGroup {
	var active []ai.CommitGroup
	for i, c := range m.commits {
		if !m.deleted[i] {
			active = append(active, c)
		}
	}
	return active
}

func (m ReviewModel) View() string {
	var b strings.Builder

	active := len(m.commits) - len(m.deleted)
	_ = active

	b.WriteString(headerStyle.Render("  Proposed commits") + "\n")
	b.WriteString(countStyle.Render(fmt.Sprintf("  %d commits, %d active", len(m.commits), len(m.commits)-len(m.deleted))) + "\n\n")

	for i, commit := range m.commits {
		prefix := "  "
		isSelected := i == m.cursor
		isDeleted := m.deleted[i]

		if isSelected {
			prefix = "▶ "
		}

		var msgLine string
		if isDeleted {
			msgLine = prefix + deletedStyle.Render(commit.Message)
		} else if isSelected && m.editing {
			msgLine = prefix + m.input.View()
		} else if isSelected {
			msgLine = prefix + selectedStyle.Render(commit.Message)
		} else {
			msgLine = prefix + normalStyle.Render(commit.Message)
		}

		b.WriteString(msgLine + "\n")

		for _, f := range commit.Files {
			if isDeleted {
				b.WriteString(fileStyle.Render(deletedStyle.Render(f)) + "\n")
			} else {
				b.WriteString(fileStyle.Render(f) + "\n")
			}
		}
		b.WriteString("\n")
	}

	help := "  ↑/↓ navigate  •  e edit  •  d toggle drop  •  enter commit  •  q quit"
	if m.editing {
		help = "  enter confirm  •  esc cancel"
	}
	b.WriteString(helpStyle.Render(help) + "\n")

	return b.String()
}
