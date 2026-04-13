package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/finn/gitai/ai"
	"github.com/finn/gitai/git"
)

type CommitResult struct {
	Message string
	Err     error
}

type CommitDoneMsg struct {
	Index  int
	Result CommitResult
}

type ExecutingModel struct {
	commits []ai.CommitGroup
	results []CommitResult
	current int
	spinner spinner.Model
	done    bool
}

var (
	doneStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	pendingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

func NewExecutingModel(commits []ai.CommitGroup) ExecutingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return ExecutingModel{
		commits: commits,
		results: make([]CommitResult, len(commits)),
		spinner: s,
	}
}

func (m ExecutingModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runNextCommit())
}

func (m ExecutingModel) runNextCommit() tea.Cmd {
	if m.current >= len(m.commits) {
		return nil
	}
	idx := m.current
	commit := m.commits[idx]
	return func() tea.Msg {
		err := git.Add(commit.Files)
		if err == nil {
			err = git.Commit(commit.Message)
		}
		return CommitDoneMsg{Index: idx, Result: CommitResult{Message: commit.Message, Err: err}}
	}
}

func (m ExecutingModel) Update(msg tea.Msg) (ExecutingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case CommitDoneMsg:
		m.results[msg.Index] = msg.Result
		m.current++
		if m.current >= len(m.commits) {
			m.done = true
			return m, nil
		}
		return m, m.runNextCommit()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m ExecutingModel) View() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Committing...") + "\n\n")

	for i, commit := range m.commits {
		if i < m.current {
			// completed
			if m.results[i].Err != nil {
				b.WriteString("  " + errorStyle.Render("✗ "+commit.Message) + "\n")
				b.WriteString("    " + errorStyle.Render(m.results[i].Err.Error()) + "\n")
			} else {
				b.WriteString("  " + doneStyle.Render("✓ "+commit.Message) + "\n")
			}
		} else if i == m.current {
			b.WriteString("  " + m.spinner.View() + " " + commit.Message + "\n")
		} else {
			b.WriteString("  " + pendingStyle.Render("  "+commit.Message) + "\n")
		}
	}

	if m.done {
		b.WriteString(fmt.Sprintf("\n  %s\n", doneStyle.Render("All done!")))
	}

	return b.String()
}

func (m ExecutingModel) Done() bool {
	return m.done
}

func (m ExecutingModel) Results() []CommitResult {
	return m.results
}
