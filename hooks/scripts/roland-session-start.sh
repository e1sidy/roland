#!/bin/bash
# Roland session start hook — inject context at the beginning of an agent session.
# This script is materialized by `roland init` and can be customized.

echo "=== Roland Session ==="
echo "Home: ${ROLAND_HOME:-~/.roland}"
echo ""

# Show ready tasks.
if command -v slate &>/dev/null; then
  echo "=== Ready Tasks ==="
  slate ready 2>/dev/null || echo "No ready tasks"
  echo ""
fi

# Show active workspaces.
if command -v roland &>/dev/null; then
  echo "=== Active Workspaces ==="
  roland status 2>/dev/null || echo "No active workspaces"
fi
