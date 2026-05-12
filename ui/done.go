package ui

import (
	"fmt"
	"strings"
)

func DoneView(results []CommitResult) string {
	var b strings.Builder

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Err != nil {
			failCount++
		} else {
			successCount++
		}
	}

	b.WriteString(headerStyle.Render("  Done!") + "\n\n")

	for _, r := range results {
		if r.Err != nil {
			b.WriteString("  " + failStyle.Render("✗ "+r.Message) + "\n")
			b.WriteString("    " + failStyle.Render(r.Err.Error()) + "\n")
		} else {
			b.WriteString("  " + successStyle.Render("✓ "+r.Message) + "\n")
		}
	}

	b.WriteString("\n")
	if failCount > 0 {
		b.WriteString("  " + warnStyle.Render(fmt.Sprintf("✓ %d committed, ✗ %d failed", successCount, failCount)) + "\n")
	} else {
		b.WriteString("  " + successStyle.Render(fmt.Sprintf("All %d commits created successfully", successCount)) + "\n")
	}

	b.WriteString("\n" + helpStyle.Render("  Press any key to exit") + "\n")
	return b.String()
}
