package templates

import (
	"context"
	"fmt"
	"strings"

	"github.com/e1sidy/slate"
)

// Decompose suggests a subtask structure for a task based on similar completed epics.
// Finds epics with matching type/labels/title keywords and extracts their structure.
func Decompose(ctx context.Context, store *slate.Store, taskID string) (*Template, error) {
	task, err := store.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Find completed epics.
	epicType := slate.TypeEpic
	closedStatus := slate.StatusClosed
	epics, err := store.List(ctx, slate.ListParams{
		Type:   &epicType,
		Status: &closedStatus,
	})
	if err != nil {
		return nil, fmt.Errorf("list epics: %w", err)
	}

	// Score each epic by similarity to the target task.
	type scored struct {
		epic  *slate.Task
		score int
	}
	var candidates []scored
	titleWords := strings.Fields(strings.ToLower(task.Title))

	for _, epic := range epics {
		score := 0
		// Type match.
		if epic.Type == task.Type {
			score += 2
		}
		// Label overlap.
		for _, l1 := range task.Labels {
			for _, l2 := range epic.Labels {
				if l1 == l2 {
					score += 3
				}
			}
		}
		// Title keyword overlap.
		epicWords := strings.Fields(strings.ToLower(epic.Title))
		for _, w1 := range titleWords {
			if len(w1) < 4 {
				continue // skip short words
			}
			for _, w2 := range epicWords {
				if w1 == w2 {
					score++
				}
			}
		}
		if score > 0 {
			candidates = append(candidates, scored{epic, score})
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no similar completed epics found")
	}

	// Pick best match.
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	// Extract template from the best matching epic.
	return CreateFromEpic(ctx, store, best.epic.ID)
}
