#!/bin/bash
# Roland pre-compact hook — auto-checkpoint before context compaction.
# Writes a breadcrumb so the agent can resume after compaction.

TASK_ID=$(basename "$(pwd)" | grep -oE '^st-[a-z0-9]+(\.[0-9]+)*')

if [ -n "$TASK_ID" ] && command -v roland &>/dev/null; then
  # Capture current git status as checkpoint.
  DIRTY_FILES=""
  for dir in */; do
    if [ -d "$dir/.git" ] || [ -L "$dir" ]; then
      STATUS=$(cd "$dir" && git status --porcelain 2>/dev/null | head -10)
      if [ -n "$STATUS" ]; then
        DIRTY_FILES="${DIRTY_FILES}${dir}: ${STATUS}\n"
      fi
    fi
  done

  roland checkpoint --done "Auto-breadcrumb before compaction" \
    --next "Resume from this checkpoint" \
    --blockers "${DIRTY_FILES:-none}" 2>/dev/null
fi
