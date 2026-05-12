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

type LogLoadedMsg struct {
	Entries []git.LogEntry
	Err     error
}

type LogDiffLoadedMsg struct {
	Diff string
	Err  error
}

type LogSummaryMsg struct {
	Summary string
	Err     error
}

type logScreen int

const (
	logScreenList logScreen = iota
	logScreenDiff
	logScreenSummary
	logScreenAILoading
)

type LogModel struct {
	entries    []git.LogEntry
	cursor     int
	diff       string
	diffHash   string
	diffScroll int
	summary    string
	screen     logScreen
	spinner    spinner.Model
	status     string
	height     int
	wantsBack  bool
}

func (m LogModel) WantsBack() bool { return m.wantsBack }

func NewLogModel() LogModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return LogModel{spinner: s, height: 24}
}

func LoadLogCmd() tea.Cmd {
	return func() tea.Msg {
		entries, err := git.Log(50)
		return LogLoadedMsg{Entries: entries, Err: err}
	}
}

func (m LogModel) Update(msg tea.Msg) (LogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		return m, nil

	case LogLoadedMsg:
		if msg.Err != nil {
			m.status = "Error: " + msg.Err.Error()
		} else {
			m.entries = msg.Entries
		}
		return m, nil

	case LogDiffLoadedMsg:
		if msg.Err != nil {
			m.status = "Error loading diff: " + msg.Err.Error()
			m.screen = logScreenList
		} else {
			m.diff = msg.Diff
			m.diffScroll = 0
			m.screen = logScreenDiff
		}
		return m, nil

	case LogSummaryMsg:
		if msg.Err != nil {
			m.status = "AI error: " + msg.Err.Error()
			m.screen = logScreenList
		} else {
			m.summary = msg.Summary
			m.screen = logScreenSummary
		}
		return m, nil

	case spinner.TickMsg:
		if m.screen == logScreenAILoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		switch m.screen {
		case logScreenAILoading:
			return m, nil

		case logScreenDiff:
			diffLines := strings.Split(m.diff, "\n")
			pageSize := m.height - 6
			if pageSize < 5 {
				pageSize = 5
			}
			switch msg.String() {
			case "up", "k":
				if m.diffScroll > 0 {
					m.diffScroll--
				}
			case "down", "j", " ":
				if m.diffScroll < len(diffLines)-pageSize {
					m.diffScroll++
				}
			case "esc", "q":
				m.screen = logScreenList
			}
			return m, nil

		case logScreenSummary:
			m.screen = logScreenList
			return m, nil

		case logScreenList:
			switch msg.String() {
			case "esc":
				m.wantsBack = true
				return m, nil
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.entries)-1 {
					m.cursor++
				}
			case "enter":
				if len(m.entries) > 0 {
					hash := m.entries[m.cursor].Hash
					m.diffHash = m.entries[m.cursor].ShortHash
					return m, func() tea.Msg {
						diff, err := git.CommitDiff(hash)
						return LogDiffLoadedMsg{Diff: diff, Err: err}
					}
				}
			case "a":
				if len(m.entries) == 0 {
					return m, nil
				}
				m.screen = logScreenAILoading
				entries := m.entries
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					var lines []string
					for _, e := range entries {
						lines = append(lines, fmt.Sprintf("%s %s (%s)", e.ShortHash, e.Message, e.DateStr))
					}
					summary, err := ai.SummarizeLog(strings.Join(lines, "\n"))
					return LogSummaryMsg{Summary: summary, Err: err}
				})
			}
		}
	}
	return m, nil
}

func (m LogModel) View() string {
	var b strings.Builder

	switch m.screen {
	case logScreenAILoading:
		b.WriteString(headerStyle.Render("  Log") + "\n\n")
		b.WriteString("  " + m.spinner.View() + " Summarizing with AI...\n")
		return b.String()

	case logScreenSummary:
		b.WriteString(headerStyle.Render("  AI Summary") + "\n\n")
		for _, line := range strings.Split(m.summary, "\n") {
			b.WriteString("  " + aiStyle.Render(line) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("  press any key to go back") + "\n")
		return b.String()

	case logScreenDiff:
		b.WriteString(headerStyle.Render("  Commit "+m.diffHash) + "\n\n")
		lines := strings.Split(m.diff, "\n")
		pageSize := m.height - 6
		if pageSize < 5 {
			pageSize = 5
		}
		start := m.diffScroll
		end := start + pageSize
		if end > len(lines) {
			end = len(lines)
		}
		for _, line := range lines[start:end] {
			styled := colorDiffLine(line)
			b.WriteString("  " + styled + "\n")
		}
		b.WriteString("\n" + helpStyle.Render(fmt.Sprintf(
			"  ↑/↓ scroll  •  esc back  [%d/%d]", m.diffScroll+1, len(lines),
		)) + "\n")
		return b.String()

	default:
		b.WriteString(headerStyle.Render("  Log") + "\n\n")
		if len(m.entries) == 0 && m.status == "" {
			b.WriteString(dimStyle.Render("  Loading...") + "\n")
		} else if m.status != "" {
			b.WriteString(statusErrStyle.Render("  "+m.status) + "\n")
		} else {
			for i, e := range m.entries {
				isSelected := i == m.cursor
				prefix := "  "
				if isSelected {
					prefix = "▶ "
				}
				hash := hashStyle.Render(e.ShortHash)
				date := dateStyle.Render(fmt.Sprintf("%-14s", e.DateStr))
				author := authorStyle.Render(fmt.Sprintf("%-12s", truncate(e.Author, 12)))
				var msg string
				if isSelected {
					msg = selectedStyle.Render(truncate(e.Message, 50))
				} else {
					msg = normalStyle.Render(truncate(e.Message, 50))
				}
				b.WriteString(fmt.Sprintf("%s%s  %s  %s  %s\n", prefix, hash, date, author, msg))
			}
		}
		b.WriteString("\n" + helpStyle.Render("  ↑/↓ navigate  •  enter view diff  •  a AI summary  •  esc back") + "\n")
	}

	return b.String()
}

func colorDiffLine(line string) string {
	if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(line)
	}
	if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(line)
	}
	if strings.HasPrefix(line, "@@") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("67")).Render(line)
	}
	if strings.HasPrefix(line, "commit ") || strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("178")).Render(line)
	}
	return dimStyle.Render(line)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
