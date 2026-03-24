package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/e1sidy/slate"
)

// DecisionEntry represents a single decision extracted from a checkpoint.
type DecisionEntry struct {
	Text    string    `json:"text"`
	TaskID  string    `json:"task_id"`
	Persona string    `json:"persona"`
	Date    time.Time `json:"date"`
}

// IndexDecisions extracts all decisions from checkpoints into a searchable list.
func IndexDecisions(ctx context.Context, store *slate.Store, params AnalyzeParams) ([]DecisionEntry, error) {
	// Get all tasks (not just closed — decisions are valuable even from in-progress tasks).
	tasks, err := store.List(ctx, slate.ListParams{})
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	var entries []DecisionEntry
	for _, t := range tasks {
		persona := getPersonaAttr(ctx, store, t.ID)
		if params.Persona != "" && persona != params.Persona {
			continue
		}

		checkpoints, err := store.ListCheckpoints(ctx, t.ID)
		if err != nil {
			continue
		}

		for _, cp := range checkpoints {
			if cp.Decisions == "" {
				continue
			}
			if params.Since != nil && cp.CreatedAt.Before(*params.Since) {
				continue
			}

			for _, line := range splitLines(cp.Decisions) {
				line = strings.TrimSpace(line)
				line = strings.TrimLeft(line, "-*•> ")
				if line == "" {
					continue
				}
				entries = append(entries, DecisionEntry{
					Text:    line,
					TaskID:  t.ID,
					Persona: persona,
					Date:    cp.CreatedAt,
				})
			}
		}
	}

	// Sort by date descending (most recent first).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.After(entries[j].Date)
	})

	return entries, nil
}

// SearchDecisions filters decision entries by keyword (case-insensitive).
func SearchDecisions(entries []DecisionEntry, query string) []DecisionEntry {
	if query == "" {
		return entries
	}
	query = strings.ToLower(query)
	var results []DecisionEntry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Text), query) {
			results = append(results, e)
		}
	}
	return results
}

// FilterByTask filters decisions to a specific task.
func FilterByTask(entries []DecisionEntry, taskID string) []DecisionEntry {
	var results []DecisionEntry
	for _, e := range entries {
		if e.TaskID == taskID {
			results = append(results, e)
		}
	}
	return results
}

// FilterByPersona filters decisions to a specific persona.
func FilterByPersona(entries []DecisionEntry, persona string) []DecisionEntry {
	var results []DecisionEntry
	for _, e := range entries {
		if e.Persona == persona {
			results = append(results, e)
		}
	}
	return results
}

// DecisionIndexPath returns the path to the decisions index file.
func DecisionIndexPath(home string) string {
	return filepath.Join(home, "decisions.json")
}

// SaveDecisionIndex writes the decision index to disk.
func SaveDecisionIndex(home string, entries []DecisionEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal decisions: %w", err)
	}
	return os.WriteFile(DecisionIndexPath(home), data, 0o644)
}

// LoadDecisionIndex reads the decision index from disk.
// Returns nil, nil if the file doesn't exist.
func LoadDecisionIndex(home string) ([]DecisionEntry, error) {
	data, err := os.ReadFile(DecisionIndexPath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read decisions: %w", err)
	}
	var entries []DecisionEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse decisions: %w", err)
	}
	return entries, nil
}
