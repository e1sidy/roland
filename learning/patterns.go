// Package learning provides pattern extraction and persona enrichment
// from completed task data (checkpoints, events, close reasons).
package learning

import (
	"sort"
	"strings"
	"unicode"

	"github.com/e1sidy/slate"
)

// Pattern represents a recurring theme found across multiple tasks.
type Pattern struct {
	Text        string   `json:"text"`
	Category    string   `json:"category"` // "decision", "blocker", "co_change", "close_reason"
	Occurrences int      `json:"occurrences"`
	TaskIDs     []string `json:"task_ids"`
	Persona     string   `json:"persona"`
}

// ExtractDecisions finds recurring decision phrases from checkpoints.
func ExtractDecisions(checkpoints []*slate.Checkpoint, taskPersonas map[string]string) []Pattern {
	phrases := make(map[string][]string) // normalized phrase → task IDs
	for _, cp := range checkpoints {
		if cp.Decisions == "" {
			continue
		}
		for _, line := range splitLines(cp.Decisions) {
			norm := normalizeText(line)
			if norm == "" || len(norm) < 10 {
				continue
			}
			phrases[norm] = appendUnique(phrases[norm], cp.TaskID)
		}
	}
	return phrasesToPatterns(phrases, "decision", taskPersonas)
}

// ExtractBlockers finds recurring blocker themes from checkpoints.
func ExtractBlockers(checkpoints []*slate.Checkpoint, taskPersonas map[string]string) []Pattern {
	phrases := make(map[string][]string)
	for _, cp := range checkpoints {
		if cp.Blockers == "" {
			continue
		}
		for _, line := range splitLines(cp.Blockers) {
			norm := normalizeText(line)
			if norm == "" || len(norm) < 10 {
				continue
			}
			phrases[norm] = appendUnique(phrases[norm], cp.TaskID)
		}
	}
	return phrasesToPatterns(phrases, "blocker", taskPersonas)
}

// ExtractCoChanges detects which file paths frequently change together
// by analyzing checkpoint file lists.
func ExtractCoChanges(checkpoints []*slate.Checkpoint, taskPersonas map[string]string) []Pattern {
	// Count file co-occurrence pairs across tasks.
	pairCounts := make(map[string][]string) // "fileA + fileB" → task IDs
	for _, cp := range checkpoints {
		if len(cp.Files) < 2 {
			continue
		}
		// Generate pairs.
		files := cp.Files
		sort.Strings(files)
		for i := 0; i < len(files); i++ {
			for j := i + 1; j < len(files); j++ {
				// Only track cross-directory pairs (same dir is obvious).
				dirA := dirOf(files[i])
				dirB := dirOf(files[j])
				if dirA == dirB {
					continue
				}
				key := files[i] + " + " + files[j]
				pairCounts[key] = appendUnique(pairCounts[key], cp.TaskID)
			}
		}
	}
	return phrasesToPatterns(pairCounts, "co_change", taskPersonas)
}

// ExtractCloseReasons finds recurring close reason patterns across tasks.
func ExtractCloseReasons(tasks []*slate.Task, taskPersonas map[string]string) []Pattern {
	phrases := make(map[string][]string)
	for _, t := range tasks {
		if t.CloseReason == "" {
			continue
		}
		norm := normalizeText(t.CloseReason)
		if norm == "" || len(norm) < 5 {
			continue
		}
		phrases[norm] = appendUnique(phrases[norm], t.ID)
	}
	return phrasesToPatterns(phrases, "close_reason", taskPersonas)
}

// FindRecurring filters patterns to only those with >= minOccurrences.
func FindRecurring(patterns []Pattern, minOccurrences int) []Pattern {
	var result []Pattern
	for _, p := range patterns {
		if p.Occurrences >= minOccurrences {
			result = append(result, p)
		}
	}
	// Sort by occurrences descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Occurrences > result[j].Occurrences
	})
	return result
}

// GroupByPersona groups patterns by their persona field.
func GroupByPersona(patterns []Pattern) map[string][]Pattern {
	groups := make(map[string][]Pattern)
	for _, p := range patterns {
		persona := p.Persona
		if persona == "" {
			persona = "unknown"
		}
		groups[persona] = append(groups[persona], p)
	}
	return groups
}

// --- Helpers ---

func normalizeText(s string) string {
	// Lowercase, strip leading bullet/dash markers, trim whitespace.
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "-*•> ")
	s = strings.ToLower(s)
	// Remove punctuation at end.
	s = strings.TrimRightFunc(s, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSpace(r)
	})
	return s
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func appendUnique(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func dirOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// phrasesToPatterns converts phrase→taskIDs map into Pattern slice.
func phrasesToPatterns(phrases map[string][]string, category string, taskPersonas map[string]string) []Pattern {
	var patterns []Pattern
	for text, taskIDs := range phrases {
		// Determine persona: most common persona among the tasks.
		persona := mostCommonPersona(taskIDs, taskPersonas)
		patterns = append(patterns, Pattern{
			Text:        text,
			Category:    category,
			Occurrences: len(taskIDs),
			TaskIDs:     taskIDs,
			Persona:     persona,
		})
	}
	return patterns
}

func mostCommonPersona(taskIDs []string, taskPersonas map[string]string) string {
	counts := make(map[string]int)
	for _, id := range taskIDs {
		p := taskPersonas[id]
		if p != "" {
			counts[p]++
		}
	}
	var best string
	var bestCount int
	for p, c := range counts {
		if c > bestCount {
			best = p
			bestCount = c
		}
	}
	return best
}
