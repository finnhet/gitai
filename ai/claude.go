package ai

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type CommitGroup struct {
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

const systemPrompt = `You are a git commit grouping assistant. Your ONLY output must be valid JSON — no markdown, no explanation, no code fences. Analyze the provided git diff and group the changes into logical, cohesive commits. Each commit must follow the Conventional Commits format (feat:, fix:, refactor:, chore:, docs:, style:, test:, etc.).

Return a JSON array in exactly this shape:
[
  {
    "message": "feat: short description",
    "files": ["path/to/file1", "path/to/file2"]
  }
]

Rules:
- Group related changes together (same feature, same bug fix, same refactor)
- Each file must appear exactly once across all groups
- Messages must be concise (under 72 chars), imperative mood
- Only output the JSON array, nothing else
- Output commit messages in the language the code is if they want different they tell`

func AnalyzeDiff(diff string) ([]CommitGroup, string, error) {
	prompt := systemPrompt + "\n\nHere is the git diff to analyze:\n\n" + diff

	cmd := exec.Command("claude", "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		raw := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			raw = string(exitErr.Stderr)
		}
		return nil, raw, fmt.Errorf("claude CLI failed: %w", err)
	}

	raw := strings.TrimSpace(string(out))

	// Strip markdown code fences if present despite instructions
	raw = stripCodeFences(raw)

	var groups []CommitGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil {
		return nil, raw, fmt.Errorf("JSON parse error: %w", err)
	}

	return groups, raw, nil
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 2 {
			// Remove first and last line (the fences)
			lines = lines[1 : len(lines)-1]
			s = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(s)
}
