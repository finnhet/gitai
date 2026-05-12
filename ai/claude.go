package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CommitGroup struct {
	Message string   `json:"message"`
	Files   []string `json:"files"`
}

const ollamaURL = "http://localhost:11434/api/generate"
const ollamaModel = "qwen3:0.6b"

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func query(prompt string) (string, error) {
	body, err := json.Marshal(ollamaRequest{
		Model:  ollamaModel,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse ollama response: %w", err)
	}

	result := strings.TrimSpace(ollamaResp.Response)
	result = stripThinkBlocks(result)
	return result, nil
}

func AnalyzeDiff(summary string) ([]CommitGroup, string, error) {
	const systemPrompt = `You are a git commit grouping assistant. Output ONLY valid JSON, no markdown or explanation.

You are given a list of changed and new files. Group them into multiple logical commits using Conventional Commits (feat:, fix:, refactor:, chore:, docs:, style:, test:).

Output format:
[{"message":"feat: short description","files":["path/to/file"]},{"message":"fix: another thing","files":["other/file"]}]

Rules:
- Every file listed must appear in exactly one commit
- ALWAYS create multiple commits when files serve different purposes
- New (untracked) files must be included using their exact path
- Messages under 72 chars, imperative mood
- Prefer more commits over fewer — one commit per logical concern`

	prompt := systemPrompt + "\n\nChanges:\n" + summary

	raw, err := query(prompt)
	if err != nil {
		return nil, "", err
	}
	raw = stripCodeFences(raw)

	var groups []CommitGroup
	if err := json.Unmarshal([]byte(raw), &groups); err != nil {
		return nil, raw, fmt.Errorf("JSON parse error: %w", err)
	}
	return groups, raw, nil
}

func SuggestBranchName(description string) (string, error) {
	prompt := `Output ONLY a git branch name in kebab-case, max 40 chars. No explanation. Examples: "add login" → feature/add-login, "fix crash" → fix/crash. Branch name for: ` + description

	result, err := query(prompt)
	if err != nil {
		return "", err
	}
	// Take first token only and clean up
	result = strings.Fields(strings.TrimSpace(result))[0]
	result = strings.Trim(result, "`\"'")
	result = strings.ReplaceAll(result, " ", "-")
	return result, nil
}

func SummarizeLog(logText string) (string, error) {
	prompt := "Summarize these git commits in 2-3 sentences. Be concise.\n\n" + logText

	return query(prompt)
}

func stripThinkBlocks(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(s, "</think>")
		if end == -1 {
			s = s[:start]
			break
		}
		s = s[:start] + s[end+8:]
	}
	return strings.TrimSpace(s)
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 2 {
			lines = lines[1 : len(lines)-1]
			s = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(s)
}
