package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/finn/gitai/ai"
	"github.com/finn/gitai/git"
	"github.com/finn/gitai/ui"
)

type screen int

const (
	screenLoading screen = iota
	screenReview
	screenExecuting
	screenDone
	screenError
)

type analyzeResultMsg struct {
	commits []ai.CommitGroup
	raw     string
	err     error
}

type model struct {
	screen    screen
	loading   ui.LoadingModel
	review    ui.ReviewModel
	executing ui.ExecutingModel
	results   []ui.CommitResult
	errMsg    string
	rawOutput string
}

func initialModel() model {
	return model{
		screen:  screenLoading,
		loading: ui.NewLoadingModel(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loading.Init(), runAnalysis())
}

func runAnalysis() tea.Cmd {
	return func() tea.Msg {
		diff, err := git.Diff()
		if err != nil {
			return analyzeResultMsg{err: err}
		}
		if diff == "" {
			return analyzeResultMsg{err: fmt.Errorf("no changes detected")}
		}
		commits, raw, err := ai.AnalyzeDiff(diff)
		return analyzeResultMsg{commits: commits, raw: raw, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.screen {
		case screenLoading:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		case screenReview:
			if m.review.IsEditing() {
				// Let review handle it
				break
			}
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "enter":
				active := m.review.ActiveCommits()
				if len(active) == 0 {
					return m, tea.Quit
				}
				exec := ui.NewExecutingModel(active)
				m.screen = screenExecuting
				m.executing = exec
				return m, m.executing.Init()
			}
		case screenExecuting:
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		case screenDone, screenError:
			return m, tea.Quit
		}

	case analyzeResultMsg:
		if msg.err != nil {
			m.screen = screenError
			m.errMsg = msg.err.Error()
			m.rawOutput = msg.raw
			return m, nil
		}
		m.review = ui.NewReviewModel(msg.commits)
		m.screen = screenReview
		return m, nil

	case ui.CommitDoneMsg:
		var cmd tea.Cmd
		m.executing, cmd = m.executing.Update(msg)
		if m.executing.Done() {
			m.results = m.executing.Results()
			m.screen = screenDone
		}
		return m, cmd
	}

	// Delegate updates to active screen
	switch m.screen {
	case screenLoading:
		var cmd tea.Cmd
		m.loading, cmd = m.loading.Update(msg)
		return m, cmd

	case screenReview:
		var cmd tea.Cmd
		m.review, cmd = m.review.Update(msg)
		return m, cmd

	case screenExecuting:
		var cmd tea.Cmd
		m.executing, cmd = m.executing.Update(msg)
		if m.executing.Done() {
			m.results = m.executing.Results()
			m.screen = screenDone
		}
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenLoading:
		return m.loading.View()
	case screenReview:
		return m.review.View()
	case screenExecuting:
		return m.executing.View()
	case screenDone:
		return ui.DoneView(m.results)
	case screenError:
		out := "\n  Error: " + m.errMsg + "\n"
		if m.rawOutput != "" {
			out += "\n  Raw output from claude:\n\n" + m.rawOutput + "\n"
		}
		out += "\n  Press any key to exit\n"
		return out
	}
	return ""
}

func main() {
	hasChanges, err := git.HasChanges()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !hasChanges {
		fmt.Println("No uncommitted changes found.")
		os.Exit(0)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
