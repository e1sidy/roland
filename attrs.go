package roland

import (
	"context"
	"fmt"

	"github.com/e1sidy/slate"
)

// Custom attribute keys used by Roland in Slate.
const (
	// AttrRepos stores a JSON array of repo names touched by a task.
	AttrRepos = "repos"

	// AttrPersonaUsed stores the persona name assigned to a task.
	AttrPersonaUsed = "persona_used"

	// AttrReviewStatus tracks review state: pending, approved, changes_requested.
	AttrReviewStatus = "review_status"

	// AttrSessionCount tracks how many agent sessions have worked on a task.
	AttrSessionCount = "session_count"
)

// rolandAttrs defines the custom attributes Roland needs in Slate.
var rolandAttrs = []struct {
	Key  string
	Type slate.AttrType
	Desc string
}{
	{AttrRepos, slate.AttrObject, "JSON array of repo names this task touches"},
	{AttrPersonaUsed, slate.AttrString, "Persona assigned to this task (e.g., builder, researcher)"},
	{AttrReviewStatus, slate.AttrString, "Review status: pending, approved, changes_requested"},
	{AttrSessionCount, slate.AttrString, "Number of agent sessions that have worked on this task"},
}

// EnsureAttrs defines all Roland-specific custom attributes in Slate.
// This is idempotent — safe to call on every pickup or init.
func EnsureAttrs(ctx context.Context, store *slate.Store) error {
	for _, a := range rolandAttrs {
		// Check if already defined.
		if _, err := store.GetAttrDef(ctx, a.Key); err == nil {
			continue
		}
		if err := store.DefineAttr(ctx, a.Key, a.Type, a.Desc); err != nil {
			return fmt.Errorf("define attr %q: %w", a.Key, err)
		}
	}
	return nil
}
