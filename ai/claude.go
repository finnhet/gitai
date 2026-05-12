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

func AnalyzeDiff(diff string) ([]CommitGroup, string, error) {
	prompt := systemPrompt + "\n\nHere is the git diff to analyze:\n\n" + diff

	body, err := json.Marshal(ollamaRequest{
		Model:  ollamaModel,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, string(respBody), fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, string(respBody), fmt.Errorf("failed to parse ollama response: %w", err)
	}

	raw := strings.TrimSpace(ollamaResp.Response)
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
			lines = lines[1 : len(lines)-1]
			s = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(s)
}
