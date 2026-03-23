package hooks

import "fmt"

// builtinHooks returns the 5 built-in hooks.
func builtinHooks() []*Hook {
	return []*Hook{
		slateInstructionsHook(),
		slateReadyTasksHook(),
		rolandInstructionsHook(),
		rolandReposHook(),
		rolandTaskContextHook(),
	}
}

// slateInstructionsHook injects a Slate CLI cheatsheet into agent sessions.
func slateInstructionsHook() *Hook {
	return &Hook{
		Name:    "slate-instructions",
		Source:  SourceHome,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return claudeStaticScript("slate-instructions", slateInstructionsContext)
		},
		OpenCodeSnippet: func(ctx HookContext) string {
			return jsStaticSnippet("slate-instructions", slateInstructionsContext)
		},
	}
}

// slateReadyTasksHook injects the output of `slate ready` into agent sessions.
func slateReadyTasksHook() *Hook {
	return &Hook{
		Name:    "slate-ready-tasks",
		Source:  SourceHome,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return claudeDynamicScript("slate-ready-tasks", "slate ready 2>/dev/null || echo 'No ready tasks'")
		},
		OpenCodeSnippet: func(ctx HookContext) string {
			return jsDynamicSnippet("slate-ready-tasks", "slate ready 2>/dev/null || echo 'No ready tasks'")
		},
	}
}

// rolandInstructionsHook injects a Roland CLI cheatsheet into agent sessions.
func rolandInstructionsHook() *Hook {
	return &Hook{
		Name:    "roland-instructions",
		Source:  SourceHome,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return claudeStaticScript("roland-instructions", rolandInstructionsContext)
		},
		OpenCodeSnippet: func(ctx HookContext) string {
			return jsStaticSnippet("roland-instructions", rolandInstructionsContext)
		},
	}
}

// rolandReposHook injects the registered repo list into agent sessions.
func rolandReposHook() *Hook {
	return &Hook{
		Name:    "roland-repos",
		Source:  SourceHome,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			return claudeDynamicScript("roland-repos", "roland repo list 2>/dev/null || echo 'No repos registered'")
		},
		OpenCodeSnippet: func(ctx HookContext) string {
			return jsDynamicSnippet("roland-repos", "roland repo list 2>/dev/null || echo 'No repos registered'")
		},
	}
}

// rolandTaskContextHook injects current task + latest checkpoint for session continuity.
func rolandTaskContextHook() *Hook {
	return &Hook{
		Name:    "roland-task-context",
		Source:  SourceTask,
		Event:   "SessionStart",
		Matcher: "*",
		ClaudeScript: func(ctx HookContext) string {
			// Detect task from current dir, show context + latest checkpoint.
			cmd := fmt.Sprintf(`TASK_ID=$(basename "$(pwd)" | grep -oE '^st-[a-z0-9]+(\.[0-9]+)*')
if [ -n "$TASK_ID" ]; then
  echo "=== Current Task ==="
  slate show "$TASK_ID" 2>/dev/null || echo "Task $TASK_ID not found"
  echo ""
  echo "=== Latest Checkpoint ==="
  slate checkpoints "$TASK_ID" 2>/dev/null | tail -20 || echo "No checkpoints"
else
  echo "No active task detected"
fi`)
			return claudeDynamicScript("roland-task-context", cmd)
		},
		OpenCodeSnippet: func(ctx HookContext) string {
			return jsDynamicSnippet("roland-task-context",
				`TASK_ID=$(basename "$(pwd)" | grep -oE '^st-[a-z0-9]+(\.[0-9]+)*') && [ -n "$TASK_ID" ] && slate show "$TASK_ID" 2>/dev/null && slate checkpoints "$TASK_ID" 2>/dev/null | tail -20`)
		},
	}
}

// --- Content strings ---

const slateInstructionsContext = `## Slate Commands

slate show <id>                    — Show task details
slate list [--status open]         — List tasks
slate create "title" [--type bug]  — Create task
slate update <id> --priority 1     — Update task
slate close <id> --reason "done"   — Close task
slate checkpoint <id> --done "..." — Add checkpoint
slate dep add <from> <to>          — Add dependency
slate search <query>               — Search tasks
slate ready                        — Show unblocked tasks
slate blocked                      — Show blocked tasks`

const rolandInstructionsContext = `## Roland Commands

roland checkpoint --done "..." --next "..."  — Record progress
roland ship [--dry-run]                      — Push + create PRs
roland ship --repo <name>                    — Ship specific repo
roland status                                — Show active tasks
roland worktree add <repo> <branch>          — Add worktree
roland worktree list <repo>                  — List worktrees`

// --- Script generators ---

// claudeStaticScript generates a bash script that outputs static content.
func claudeStaticScript(name, content string) string {
	return fmt.Sprintf(`#!/bin/bash
cat << 'HOOK_EOF'
%s
HOOK_EOF
`, content)
}

// claudeDynamicScript generates a bash script that runs a command.
func claudeDynamicScript(name, command string) string {
	return fmt.Sprintf(`#!/bin/bash
%s
`, command)
}

// jsStaticSnippet generates a JS snippet that returns static content.
func jsStaticSnippet(name, content string) string {
	return fmt.Sprintf(`// %s
return %q;`, name, content)
}

// jsDynamicSnippet generates a JS snippet that runs a shell command.
func jsDynamicSnippet(name, command string) string {
	return fmt.Sprintf(`// %s
const { execSync } = require('child_process');
try {
  return execSync(%q, { encoding: 'utf-8', timeout: 10000 });
} catch(e) {
  return '';
}`, name, command)
}
