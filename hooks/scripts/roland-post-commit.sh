#!/bin/bash
# Roland post-commit hook — auto-comment on task when commit message contains task ID.
# Place this in .git/hooks/post-commit or use with Claude Code hooks.

COMMIT_MSG=$(git log -1 --format=%B 2>/dev/null)
TASK_ID=$(echo "$COMMIT_MSG" | grep -oE 'st-[a-z0-9]+(\.[0-9]+)*' | head -1)

if [ -n "$TASK_ID" ] && command -v slate &>/dev/null; then
  HASH=$(git log -1 --format=%h 2>/dev/null)
  SUBJECT=$(git log -1 --format=%s 2>/dev/null)
  slate comment add "$TASK_ID" "commit $HASH: $SUBJECT" --author "git-hook" 2>/dev/null
fi
