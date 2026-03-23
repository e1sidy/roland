package skill

// MatchContext provides the current task context for skill matching.
type MatchContext struct {
	Persona  string   // Active persona name (e.g., "builder").
	TaskType string   // Task type (e.g., "bug", "feature").
	Labels   []string // Task labels/tags.
}

// Match returns true if the skill should be auto-injected for the given context.
//
// Matching uses OR logic across non-empty dimensions:
//   - If skill has Personas and ctx.Persona matches any → true
//   - If skill has TaskTypes and ctx.TaskType matches any → true
//   - If skill has Tags and any ctx.Label matches any → true
//   - If all dimensions are empty → false (manual-only skill)
func Match(entry *SkillEntry, ctx *MatchContext) bool {
	if len(entry.Personas) > 0 && ctx.Persona != "" {
		for _, p := range entry.Personas {
			if p == ctx.Persona {
				return true
			}
		}
	}
	if len(entry.TaskTypes) > 0 && ctx.TaskType != "" {
		for _, t := range entry.TaskTypes {
			if t == ctx.TaskType {
				return true
			}
		}
	}
	if len(entry.Tags) > 0 && len(ctx.Labels) > 0 {
		if intersects(entry.Tags, ctx.Labels) {
			return true
		}
	}
	return false
}

// MatchAll returns the names of all skills that match the given context.
func MatchAll(sc *SkillConfig, ctx *MatchContext) []string {
	var names []string
	for name, entry := range sc.Skills {
		if Match(entry, ctx) {
			names = append(names, name)
		}
	}
	return names
}

// intersects returns true if any element appears in both slices.
func intersects(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, s := range a {
		set[s] = true
	}
	for _, s := range b {
		if set[s] {
			return true
		}
	}
	return false
}
