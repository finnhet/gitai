package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/finn/gitai/ai"
	"github.com/finn/gitai/git"
)

type BranchesLoadedMsg struct {
	Branches []git.Branch
	Err      error
}

type BranchSwitchedMsg struct {
	Branch string
	Err    error
}

type BranchCreatedMsg struct {
	Branch string
	Err    error
}

type AIBranchSuggestedMsg struct {
	Name string
	Err  error
}

type branchMode int

const (
	branchModeFilter branchMode = iota
	branchModeNew
	branchModeAILoading
)

type BranchModel struct {
	branches  []git.Branch
	filtered  []git.Branch
	cursor    int
	input     textinput.Model
	mode      branchMode
	status    string
	statusOk  bool
	spinner   spinner.Model
	wantsBack bool
}

func (m BranchModel) WantsBack() bool { return m.wantsBack }

func NewBranchModel() BranchModel {
	ti := textinput.New()
	ti.Placeholder = "filter branches..."
	ti.Focus()
	ti.CharLimit = 80

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return BranchModel{input: ti, spinner: s}
}

func LoadBranchesCmd() tea.Cmd {
	return func() tea.Msg {
		branches, err := git.Branches()
		return BranchesLoadedMsg{Branches: branches, Err: err}
	}
}

func (m BranchModel) applyFilter() []git.Branch {
	q := strings.ToLower(m.input.Value())
	if q == "" {
		return m.branches
	}
	var out []git.Branch
	for _, b := range m.branches {
		if strings.Contains(strings.ToLower(b.Name), q) {
			out = append(out, b)
		}
	}
	return out
}

func (m BranchModel) Update(msg tea.Msg) (BranchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case BranchesLoadedMsg:
		if msg.Err != nil {
			m.status = "Error: " + msg.Err.Error()
			m.statusOk = false
		} else {
			m.branches = msg.Branches
			m.filtered = m.branches
		}
		return m, nil

	case BranchSwitchedMsg:
		if msg.Err != nil {
			m.status = msg.Err.Error()
			m.statusOk = false
		} else {
			m.status = "Switched to " + msg.Branch
			m.statusOk = true
		}
		// Reload branches so current marker updates
		return m, LoadBranchesCmd()

	case BranchCreatedMsg:
		if msg.Err != nil {
			m.status = msg.Err.Error()
			m.statusOk = false
		} else {
			m.status = "Created and switched to " + msg.Branch
			m.statusOk = true
			m.mode = branchModeFilter
			m.input.Placeholder = "filter branches..."
			m.input.SetValue("")
		}
		return m, LoadBranchesCmd()

	case AIBranchSuggestedMsg:
		m.mode = branchModeNew
		if msg.Err != nil {
			m.status = "AI error: " + msg.Err.Error()
			m.statusOk = false
		} else {
			m.input.SetValue(msg.Name)
			m.status = "AI suggested — edit or press enter to create"
			m.statusOk = true
		}
		return m, nil

	case spinner.TickMsg:
		if m.mode == branchModeAILoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		switch m.mode {
		case branchModeAILoading:
			// Block input while AI is running
			return m, nil

		case branchModeNew:
			switch msg.Type {
			case tea.KeyEnter:
				name := strings.TrimSpace(m.input.Value())
				if name == "" {
					return m, nil
				}
				return m, func() tea.Msg {
					err := git.CreateBranch(name)
					return BranchCreatedMsg{Branch: name, Err: err}
				}
			case tea.KeyEsc:
				m.mode = branchModeFilter
				m.input.Placeholder = "filter branches..."
				m.input.SetValue("")
				m.filtered = m.branches
				m.status = ""
				return m, nil
			case tea.KeyCtrlA:
				// Ask AI to suggest from current input as description
				desc := strings.TrimSpace(m.input.Value())
				if desc == "" {
					m.status = "Type a description first, then press ctrl+a"
					return m, nil
				}
				m.mode = branchModeAILoading
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					name, err := ai.SuggestBranchName(desc)
					return AIBranchSuggestedMsg{Name: name, Err: err}
				})
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd

		case branchModeFilter:
			switch msg.String() {
			case "esc":
				m.wantsBack = true
				return m, nil
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down", "j":
				if m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
				return m, nil
			case "enter":
				if len(m.filtered) > 0 {
					branch := m.filtered[m.cursor]
					if branch.Current {
						m.status = "Already on " + branch.Name
						m.statusOk = true
						return m, nil
					}
					return m, func() tea.Msg {
						err := git.Checkout(branch.Name)
						return BranchSwitchedMsg{Branch: branch.Name, Err: err}
					}
				}
			case "n":
				m.mode = branchModeNew
				m.input.Placeholder = "branch name (ctrl+a for AI suggestion)..."
				m.input.SetValue("")
				m.status = ""
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.filtered = m.applyFilter()
			m.cursor = 0
			return m, cmd
		}
	}
	return m, nil
}

func (m BranchModel) View() string {
	var b strings.Builder

	switch m.mode {
	case branchModeAILoading:
		b.WriteString(headerStyle.Render("  Branches") + "\n\n")
		b.WriteString("  " + m.spinner.View() + " Asking AI for a branch name...\n")
		return b.String()
	case branchModeNew:
		b.WriteString(headerStyle.Render("  New Branch") + "\n\n")
		b.WriteString("  " + m.input.View() + "\n\n")
		b.WriteString(helpStyle.Render("  enter create  •  ctrl+a AI suggest  •  esc cancel") + "\n")
	default:
		b.WriteString(headerStyle.Render("  Branches") + "\n\n")
		b.WriteString("  " + m.input.View() + "\n\n")

		if len(m.filtered) == 0 && len(m.branches) == 0 {
			b.WriteString(dimStyle.Render("  Loading...") + "\n")
		} else if len(m.filtered) == 0 {
			b.WriteString(dimStyle.Render("  No branches match") + "\n")
		} else {
			for i, branch := range m.filtered {
				isSelected := i == m.cursor
				prefix := "  "
				if isSelected {
					prefix = "▶ "
				}

				var name string
				if branch.Current {
					name = selectedStyle.Render("* " + branch.Name)
				} else if branch.Remote {
					name = remoteStyle.Render(branch.Name)
				} else if isSelected {
					name = normalStyle.Render(branch.Name)
				} else {
					name = dimStyle.Render(branch.Name)
				}

				b.WriteString(prefix + name + "\n")
			}
		}
		b.WriteString("\n" + helpStyle.Render("  ↑/↓ navigate  •  enter checkout  •  n new branch  •  esc back") + "\n")
	}

	if m.status != "" {
		b.WriteString("\n  ")
		if m.statusOk {
			b.WriteString(statusOkStyle.Render(m.status))
		} else {
			b.WriteString(statusErrStyle.Render(m.status))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m BranchModel) CurrentBranchName() string {
	for _, b := range m.branches {
		if b.Current {
			return b.Name
		}
	}
	return ""
}

