package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/finn/gitai/ai"
	"github.com/finn/gitai/git"
	"github.com/finn/gitai/ui"
)

type screen int

const (
	screenHome screen = iota
	screenLoading
	screenReview
	screenExecuting
	screenDone
	screenError
	screenBranches
	screenLog
	screenPush
)

type analyzeResultMsg struct {
	commits []ai.CommitGroup
	raw     string
	err     error
}

type pushPullDoneMsg struct {
	op  string
	err error
}

type model struct {
	screen    screen
	width     int
	height    int
	// commit flow
	loading   ui.LoadingModel
	review    ui.ReviewModel
	executing ui.ExecutingModel
	results   []ui.CommitResult
	errMsg    string
	rawOutput string
	// home
	home ui.HomeModel
	// branch switcher
	branches ui.BranchModel
	// log viewer
	log ui.LogModel
	// push/pull
	pushOp      string
	pushMsg     string
	pushOk      bool
	pushSpinner spinner.Model
	pushing     bool
}

func initialModel() model {
	ps := spinner.New()
	ps.Spinner = spinner.Dot
	ps.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		screen:      screenHome,
		home:        ui.NewHomeModel(),
		pushSpinner: ps,
		height:      24,
	}
}

func (m model) Init() tea.Cmd {
	return loadHomeData()
}

func loadHomeData() tea.Cmd {
	return func() tea.Msg {
		branch, err := git.CurrentBranch()
		if err != nil {
			return ui.HomeReadyMsg{Err: err}
		}
		changes, err := git.ChangeCount()
		if err != nil {
			return ui.HomeReadyMsg{Branch: branch, Err: err}
		}
		return ui.HomeReadyMsg{Branch: branch, Changes: changes}
	}
}

func runAnalysis() tea.Cmd {
	return func() tea.Msg {
		summary, err := git.ChangeSummary()
		if err != nil {
			return analyzeResultMsg{err: err}
		}
		commits, raw, err := ai.AnalyzeDiff(summary)
		return analyzeResultMsg{commits: commits, raw: raw, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		var cmd tea.Cmd
		m.log, cmd = m.log.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)

	case ui.HomeReadyMsg:
		var cmd tea.Cmd
		m.home, cmd = m.home.Update(msg)
		return m, cmd

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

	case ui.BranchesLoadedMsg, ui.BranchSwitchedMsg, ui.BranchCreatedMsg, ui.AIBranchSuggestedMsg:
		var cmd tea.Cmd
		m.branches, cmd = m.branches.Update(msg)
		return m, cmd

	case ui.LogLoadedMsg, ui.LogDiffLoadedMsg, ui.LogSummaryMsg:
		var cmd tea.Cmd
		m.log, cmd = m.log.Update(msg)
		return m, cmd

	case pushPullDoneMsg:
		m.pushing = false
		m.pushOk = msg.err == nil
		if msg.err != nil {
			m.pushMsg = msg.err.Error()
		}
		return m, nil

	case spinner.TickMsg:
		if m.pushing {
			var cmd tea.Cmd
			m.pushSpinner, cmd = m.pushSpinner.Update(msg)
			return m, cmd
		}
		if m.screen == screenBranches {
			var cmd tea.Cmd
			m.branches, cmd = m.branches.Update(msg)
			return m, cmd
		}
		if m.screen == screenLog {
			var cmd tea.Cmd
			m.log, cmd = m.log.Update(msg)
			return m, cmd
		}
	}

	// Delegate to active sub-model
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

	case screenBranches:
		var cmd tea.Cmd
		m.branches, cmd = m.branches.Update(msg)
		return m, cmd

	case screenLog:
		var cmd tea.Cmd
		m.log, cmd = m.log.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenHome:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "c":
			m.screen = screenLoading
			m.loading = ui.NewLoadingModel()
			return m, tea.Batch(m.loading.Init(), runAnalysis())
		case "b":
			m.screen = screenBranches
			m.branches = ui.NewBranchModel()
			return m, ui.LoadBranchesCmd()
		case "l":
			m.screen = screenLog
			m.log = ui.NewLogModel()
			return m, ui.LoadLogCmd()
		case "p":
			m.screen = screenPush
			m.pushOp = "push"
			m.pushMsg = ""
			m.pushing = true
			return m, tea.Batch(m.pushSpinner.Tick, func() tea.Msg {
				err := git.Push()
				return pushPullDoneMsg{op: "push", err: err}
			})
		case "f":
			m.screen = screenPush
			m.pushOp = "pull"
			m.pushMsg = ""
			m.pushing = true
			return m, tea.Batch(m.pushSpinner.Tick, func() tea.Msg {
				err := git.Pull()
				return pushPullDoneMsg{op: "pull", err: err}
			})
		}

	case screenLoading:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case screenReview:
		if m.review.IsEditing() {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.screen = screenHome
			return m, loadHomeData()
		case "enter":
			active := m.review.ActiveCommits()
			if len(active) == 0 {
				return m, nil
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
		switch msg.String() {
		case "esc", "enter", "q":
			m.screen = screenHome
			return m, loadHomeData()
		default:
			return m, tea.Quit
		}

	case screenBranches:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		var branchCmd tea.Cmd
		m.branches, branchCmd = m.branches.Update(msg)
		if m.branches.WantsBack() {
			m.screen = screenHome
			return m, loadHomeData()
		}
		return m, branchCmd

	case screenLog:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		var logCmd tea.Cmd
		m.log, logCmd = m.log.Update(msg)
		if m.log.WantsBack() {
			m.screen = screenHome
			return m, loadHomeData()
		}
		return m, logCmd

	case screenPush:
		if !m.pushing {
			m.screen = screenHome
			return m, loadHomeData()
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	switch m.screen {
	case screenHome:
		return m.home.View()
	case screenLoading:
		return m.loading.View()
	case screenReview:
		return m.review.View()
	case screenExecuting:
		return m.executing.View()
	case screenDone:
		return ui.DoneView(m.results)
	case screenError:
		out := "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: "+m.errMsg) + "\n"
		if m.rawOutput != "" {
			out += "\n  Raw AI output:\n\n" + m.rawOutput + "\n"
		}
		out += "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Press esc to go back") + "\n"
		return out
	case screenBranches:
		return m.branches.View()
	case screenLog:
		return m.log.View()
	case screenPush:
		var out string
		opLabel := m.pushOp
		if m.pushing {
			out = "\n  " + m.pushSpinner.View() + " " + opLabel + "ing...\n"
		} else {
			if m.pushOk {
				out = "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓ "+opLabel+" complete") + "\n"
			} else {
				out = "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗ "+m.pushMsg) + "\n"
			}
			out += "\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Press any key to go back") + "\n"
		}
		return out
	}
	return ""
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
