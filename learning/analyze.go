package learning

import (
	"context"
	"fmt"
	"time"

	"github.com/e1sidy/slate"
)

// AnalyzeParams controls the learning analysis scope.
type AnalyzeParams struct {
	Since          *time.Time // only tasks closed after this date
	Persona        string     // filter by persona_used attr (empty = all)
	MinOccurrences int        // minimum occurrences for a pattern (default: 3)
}

// Analyze queries completed tasks and extracts recurring patterns from
// checkpoints, close reasons, and file co-changes.
func Analyze(ctx context.Context, store *slate.Store, params AnalyzeParams) ([]Pattern, error) {
	if params.MinOccurrences <= 0 {
		params.MinOccurrences = 3
	}

	// Query closed tasks.
	closedStatus := slate.StatusClosed
	tasks, err := store.List(ctx, slate.ListParams{Status: &closedStatus})
	if err != nil {
		return nil, fmt.Errorf("list closed tasks: %w", err)
	}

	// Filter by date and persona.
	taskPersonas := make(map[string]string) // taskID → persona
	var filtered []*slate.Task
	for _, t := range tasks {
		// Date filter.
		if params.Since != nil && t.ClosedAt != nil && t.ClosedAt.Before(*params.Since) {
			continue
		}

		// Persona filter via custom attribute.
		persona := getPersonaAttr(ctx, store, t.ID)
		taskPersonas[t.ID] = persona
		if params.Persona != "" && persona != params.Persona {
			continue
		}

		filtered = append(filtered, t)
	}

	if len(filtered) == 0 {
		return nil, nil
	}

	// Collect all checkpoints for filtered tasks.
	var allCheckpoints []*slate.Checkpoint
	for _, t := range filtered {
		cps, err := store.ListCheckpoints(ctx, t.ID)
		if err != nil {
			continue
		}
		allCheckpoints = append(allCheckpoints, cps...)
	}

	// Extract patterns from each data source.
	var allPatterns []Pattern
	allPatterns = append(allPatterns, ExtractDecisions(allCheckpoints, taskPersonas)...)
	allPatterns = append(allPatterns, ExtractBlockers(allCheckpoints, taskPersonas)...)
	allPatterns = append(allPatterns, ExtractCoChanges(allCheckpoints, taskPersonas)...)
	allPatterns = append(allPatterns, ExtractCloseReasons(filtered, taskPersonas)...)

	// Filter to recurring patterns only.
	return FindRecurring(allPatterns, params.MinOccurrences), nil
}

// getPersonaAttr reads the persona_used custom attribute for a task.
func getPersonaAttr(ctx context.Context, store *slate.Store, taskID string) string {
	attr, err := store.GetAttr(ctx, taskID, "persona_used")
	if err != nil || attr == nil {
		return ""
	}
	return attr.Value
}
