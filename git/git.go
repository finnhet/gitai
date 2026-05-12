package git

import (
	"fmt"
	"os/exec"
	"strings"
)

type Branch struct {
	Name    string
	Current bool
	Remote  bool
}

type LogEntry struct {
	Hash      string
	ShortHash string
	Author    string
	DateStr   string
	Message   string
}

func HasChanges() (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func ChangeCount() (int, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return 0, fmt.Errorf("git status failed: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, nil
	}
	return len(strings.Split(s, "\n")), nil
}

func Diff() (string, error) {
	out, err := exec.Command("git", "diff", "HEAD").Output()
	if err != nil {
		out, err = exec.Command("git", "diff", "--cached").Output()
		if err != nil {
			return "", fmt.Errorf("git diff failed: %w", err)
		}
	}
	if len(out) == 0 {
		cached, _ := exec.Command("git", "diff", "--cached").Output()
		if len(cached) > 0 {
			return string(cached), nil
		}
	}
	return string(out), nil
}

// ChangeSummary returns a clean file-per-line listing of every pending change.
// It uses git status --porcelain so ALL files are included regardless of
// whether they are staged, unstaged, or untracked.
func ChangeSummary() (string, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "", fmt.Errorf("no changes detected")
	}

	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if len(line) < 4 {
			continue
		}
		xy := line[:2]
		file := strings.TrimSpace(line[2:])
		// Handle renames: "old -> new"
		if idx := strings.Index(file, " -> "); idx != -1 {
			file = file[idx+4:]
		}
		switch {
		case xy == "??":
			b.WriteString("new file: " + file + "\n")
		case strings.Contains(xy, "D"):
			b.WriteString("deleted: " + file + "\n")
		default:
			b.WriteString("modified: " + file + "\n")
		}
	}

	if b.Len() == 0 {
		return "", fmt.Errorf("no changes detected")
	}
	return b.String(), nil
}

func Add(files []string) error {
	args := append([]string{"add", "--"}, files...)
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %s\n%w", string(out), err)
	}
	return nil
}

func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %s\n%w", string(out), err)
	}
	return nil
}

func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func Branches() ([]Branch, error) {
	out, err := exec.Command("git", "branch", "-a").Output()
	if err != nil {
		return nil, fmt.Errorf("git branch failed: %w", err)
	}
	var branches []Branch
	for _, line := range strings.Split(string(out), "\n") {
		raw := line
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.Contains(trimmed, "HEAD ->") {
			continue
		}
		b := Branch{}
		if strings.HasPrefix(raw, "* ") {
			b.Current = true
			trimmed = strings.TrimPrefix(trimmed, "* ")
		}
		if strings.HasPrefix(trimmed, "remotes/") {
			b.Remote = true
			trimmed = strings.TrimPrefix(trimmed, "remotes/")
		}
		b.Name = trimmed
		branches = append(branches, b)
	}
	return branches, nil
}

func Checkout(branch string) error {
	if strings.HasPrefix(branch, "origin/") {
		localName := strings.TrimPrefix(branch, "origin/")
		cmd := exec.Command("git", "checkout", "-b", localName, "--track", branch)
		if _, err := cmd.CombinedOutput(); err != nil {
			cmd2 := exec.Command("git", "checkout", localName)
			out2, err2 := cmd2.CombinedOutput()
			if err2 != nil {
				return fmt.Errorf("git checkout failed: %s\n%w", string(out2), err2)
			}
		}
		return nil
	}
	cmd := exec.Command("git", "checkout", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %s\n%w", string(out), err)
	}
	return nil
}

func CreateBranch(name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b failed: %s\n%w", string(out), err)
	}
	return nil
}

func Log(limit int) ([]LogEntry, error) {
	out, err := exec.Command("git", "log", fmt.Sprintf("-n%d", limit), "--pretty=format:%H|%an|%ar|%s").Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	var entries []LogEntry
	for _, line := range strings.Split(s, "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}
		hash := parts[0]
		short := hash
		if len(hash) > 7 {
			short = hash[:7]
		}
		entries = append(entries, LogEntry{
			Hash:      hash,
			ShortHash: short,
			Author:    parts[1],
			DateStr:   parts[2],
			Message:   parts[3],
		})
	}
	return entries, nil
}

func CommitDiff(hash string) (string, error) {
	out, err := exec.Command("git", "show", hash).Output()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(out), nil
}

func Push() error {
	branch, err := CurrentBranch()
	if err != nil {
		return err
	}
	cmd := exec.Command("git", "push", "-u", "origin", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %s\n%w", string(out), err)
	}
	return nil
}

func Pull() error {
	cmd := exec.Command("git", "pull")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s\n%w", string(out), err)
	}
	return nil
}
