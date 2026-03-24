package hooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/e1sidy/slate"
)

const defaultContextBudget = 4096 // tokens

// BuildContext assembles agent context within a token budget.
// Priority order: task context > blocking deps > recent comments > ready tasks > historical checkpoints.
func BuildContext(ctx context.Context, store *slate.Store, taskID string, budget int) (string, error) {
	if budget <= 0 {
		budget = defaultContextBudget
	}

	var sections []string
	remaining := budget

	// Priority 1: Current task context (always included).
	taskSection, tokens := buildTaskSection(ctx, store, taskID)
	sections = append(sections, taskSection)
	remaining -= tokens

	// Priority 2: Blocking dependencies (always included).
	depsSection, tokens := buildDepsSection(ctx, store, taskID)
	if tokens > 0 {
		sections = append(sections, depsSection)
		remaining -= tokens
	}

	// Priority 3: Recent comments (if budget allows).
	if remaining > 100 {
		commentsSection, tokens := buildCommentsSection(ctx, store, taskID, remaining)
		if tokens > 0 {
			sections = append(sections, commentsSection)
			remaining -= tokens
		}
	}

	// Priority 4: Ready tasks (if budget allows).
	if remaining > 200 {
		readySection, tokens := buildReadySection(ctx, store, remaining)
		if tokens > 0 {
			sections = append(sections, readySection)
			remaining -= tokens
		}
	}

	// Priority 5: Historical checkpoints (summarized, if budget allows).
	if remaining > 100 {
		histSection, _ := buildHistoricalCheckpoints(ctx, store, taskID, remaining)
		if histSection != "" {
			sections = append(sections, histSection)
		}
	}

	return strings.Join(sections, "\n\n"), nil
}

// EstimateTokens estimates the token count for a string (1 token ≈ 4 chars).
func EstimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// SummarizeCheckpoints compresses older checkpoints to fit within a token budget.
func SummarizeCheckpoints(checkpoints []*slate.Checkpoint, maxTokens int) string {
	if len(checkpoints) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("Previous checkpoints:\n")

	for i := len(checkpoints) - 1; i >= 0; i-- {
		cp := checkpoints[i]
		line := fmt.Sprintf("- [%s] %s", cp.CreatedAt.Format("Jan 2"), cp.Done)
		if EstimateTokens(buf.String()+line) > maxTokens {
			buf.WriteString(fmt.Sprintf("- ... and %d earlier checkpoints\n", i+1))
			break
		}
		buf.WriteString(line + "\n")
	}

	return buf.String()
}

// PruneStaleReady filters out ready tasks older than maxAge.
func PruneStaleReady(tasks []*slate.Task, maxAge time.Duration) []*slate.Task {
	cutoff := time.Now().Add(-maxAge)
	var fresh []*slate.Task
	for _, t := range tasks {
		if t.CreatedAt.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	return fresh
}

// --- Section builders ---

func buildTaskSection(ctx context.Context, store *slate.Store, taskID string) (string, int) {
	task, err := store.GetFull(ctx, taskID)
	if err != nil {
		return fmt.Sprintf("Current task: %s (error loading)", taskID), 10
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# Current Task: [%s] %s\n", task.ID, task.Title))
	if task.Description != "" {
		buf.WriteString(fmt.Sprintf("Description: %s\n", task.Description))
	}
	buf.WriteString(fmt.Sprintf("Status: %s | Priority: %s | Type: %s\n", task.Status, task.Priority, task.Type))

	// Latest checkpoint.
	cp, _ := store.LatestCheckpoint(ctx, taskID)
	if cp != nil {
		buf.WriteString(fmt.Sprintf("\nLatest checkpoint (%s):\n", cp.CreatedAt.Format("Jan 2 15:04")))
		buf.WriteString(fmt.Sprintf("  Done: %s\n", cp.Done))
		if cp.Next != "" {
			buf.WriteString(fmt.Sprintf("  Next: %s\n", cp.Next))
		}
		if cp.Blockers != "" {
			buf.WriteString(fmt.Sprintf("  Blockers: %s\n", cp.Blockers))
		}
		if cp.Decisions != "" {
			buf.WriteString(fmt.Sprintf("  Decisions: %s\n", cp.Decisions))
		}
	}

	text := buf.String()
	return text, EstimateTokens(text)
}

func buildDepsSection(ctx context.Context, store *slate.Store, taskID string) (string, int) {
	deps, err := store.ListDependents(ctx, taskID)
	if err != nil || len(deps) == 0 {
		return "", 0
	}

	var buf strings.Builder
	buf.WriteString("## Blocking Dependencies\n")
	for _, dep := range deps {
		if dep.Type != slate.Blocks {
			continue
		}
		blocker, _ := store.Get(ctx, dep.FromID)
		if blocker == nil {
			continue
		}
		buf.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", blocker.ID, blocker.Title, blocker.Status))
	}

	text := buf.String()
	return text, EstimateTokens(text)
}

func buildCommentsSection(ctx context.Context, store *slate.Store, taskID string, maxTokens int) (string, int) {
	comments, err := store.ListComments(ctx, taskID)
	if err != nil || len(comments) == 0 {
		return "", 0
	}

	var buf strings.Builder
	buf.WriteString("## Recent Comments\n")

	// Last 5 comments.
	start := 0
	if len(comments) > 5 {
		start = len(comments) - 5
	}
	for _, c := range comments[start:] {
		line := fmt.Sprintf("- [%s] %s: %s\n", c.CreatedAt.Format("Jan 2"), c.Author, c.Content)
		if EstimateTokens(buf.String()+line) > maxTokens {
			break
		}
		buf.WriteString(line)
	}

	text := buf.String()
	return text, EstimateTokens(text)
}

func buildReadySection(ctx context.Context, store *slate.Store, maxTokens int) (string, int) {
	ready, err := store.Ready(ctx, "")
	if err != nil || len(ready) == 0 {
		return "", 0
	}

	// Prune stale (older than 30 days).
	ready = PruneStaleReady(ready, 30*24*time.Hour)

	// Top 10 by priority.
	if len(ready) > 10 {
		ready = ready[:10]
	}

	var buf strings.Builder
	buf.WriteString("## Ready Tasks\n")
	for _, t := range ready {
		line := fmt.Sprintf("- [%s] %s (P%d)\n", t.ID, t.Title, t.Priority)
		if EstimateTokens(buf.String()+line) > maxTokens {
			break
		}
		buf.WriteString(line)
	}

	text := buf.String()
	return text, EstimateTokens(text)
}

func buildHistoricalCheckpoints(ctx context.Context, store *slate.Store, taskID string, maxTokens int) (string, int) {
	cps, err := store.ListCheckpoints(ctx, taskID)
	if err != nil || len(cps) <= 1 {
		return "", 0 // latest already shown in task section
	}

	// Exclude the latest (already in task section).
	older := cps[:len(cps)-1]
	text := SummarizeCheckpoints(older, maxTokens)
	return text, EstimateTokens(text)
}
