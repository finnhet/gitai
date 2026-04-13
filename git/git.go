package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func HasChanges() (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func Diff() (string, error) {
	// git diff HEAD covers both staged and unstaged changes
	out, err := exec.Command("git", "diff", "HEAD").Output()
	if err != nil {
		// If HEAD doesn't exist (initial commit), fall back to cached diff
		out, err = exec.Command("git", "diff", "--cached").Output()
		if err != nil {
			return "", fmt.Errorf("git diff failed: %w", err)
		}
	}
	if len(out) == 0 {
		// Try cached only (all staged, nothing unstaged)
		cached, _ := exec.Command("git", "diff", "--cached").Output()
		if len(cached) > 0 {
			return string(cached), nil
		}
	}
	return string(out), nil
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
