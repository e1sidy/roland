package roland

import (
	"testing"

	"github.com/e1sidy/roland/internal/testutil"
)

func TestEnsureAttrs_Idempotent(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	// First call should define all attrs.
	if err := EnsureAttrs(ctx, store); err != nil {
		t.Fatalf("EnsureAttrs (first): %v", err)
	}

	// Second call should be a no-op (idempotent).
	if err := EnsureAttrs(ctx, store); err != nil {
		t.Fatalf("EnsureAttrs (second): %v", err)
	}
}

func TestEnsureAttrs_DefinesAll(t *testing.T) {
	store, ctx := testutil.TempSlateStore(t)

	if err := EnsureAttrs(ctx, store); err != nil {
		t.Fatalf("EnsureAttrs: %v", err)
	}

	// Verify each attr is defined.
	for _, key := range []string{AttrRepos, AttrPersonaUsed, AttrReviewStatus, AttrSessionCount} {
		def, err := store.GetAttrDef(ctx, key)
		if err != nil {
			t.Errorf("GetAttrDef(%q): %v", key, err)
			continue
		}
		if def == nil {
			t.Errorf("attr %q not defined", key)
		}
	}
}
