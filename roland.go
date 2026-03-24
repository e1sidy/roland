// Package roland provides workspace orchestration for AI coding agents.
//
// Roland manages the full lifecycle of agent-driven development:
// pickup → work → checkpoint → ship → done. It integrates with Slate
// (the task layer) for state management and provides personas, hooks,
// skills, and workspace isolation via git worktrees.
package roland

import "fmt"

// AgentTool identifies a supported AI coding agent.
type AgentTool string

const (
	AgentClaude   AgentTool = "claude"
	AgentOpenCode AgentTool = "opencode"
	AgentCodex    AgentTool = "codex"
	AgentGemini   AgentTool = "gemini"
)

// validAgents lists all supported agent tools.
var validAgents = []AgentTool{AgentClaude, AgentOpenCode, AgentCodex, AgentGemini}

// IsValid returns true if the agent tool is recognized.
func (a AgentTool) IsValid() bool {
	for _, v := range validAgents {
		if a == v {
			return true
		}
	}
	return false
}

// Command returns the CLI command name used to launch this agent.
func (a AgentTool) Command() string {
	switch a {
	case AgentClaude:
		return "claude"
	case AgentOpenCode:
		return "opencode"
	case AgentCodex:
		return "codex"
	case AgentGemini:
		return "gemini"
	default:
		return string(a)
	}
}

// ValidAgentNames returns all valid agent tool names.
func ValidAgentNames() []string {
	names := make([]string, len(validAgents))
	for i, a := range validAgents {
		names[i] = string(a)
	}
	return names
}

// IDE identifies a supported code editor / IDE.
type IDE string

const (
	IDEVSCode  IDE = "vscode"
	IDECursor  IDE = "cursor"
	IDEWindsurf IDE = "windsurf"
	IDENvim    IDE = "nvim"
)

// validIDEs lists all supported IDEs.
var validIDEs = []IDE{IDEVSCode, IDECursor, IDEWindsurf, IDENvim}

// IsValid returns true if the IDE is recognized.
func (i IDE) IsValid() bool {
	for _, v := range validIDEs {
		if i == v {
			return true
		}
	}
	return false
}

// Command returns the CLI command used to open this IDE.
func (i IDE) Command() string {
	switch i {
	case IDEVSCode:
		return "code"
	case IDECursor:
		return "cursor"
	case IDEWindsurf:
		return "windsurf"
	case IDENvim:
		return "nvim"
	default:
		return string(i)
	}
}

// ValidIDENames returns all valid IDE names.
func ValidIDENames() []string {
	names := make([]string, len(validIDEs))
	for i, ide := range validIDEs {
		names[i] = string(ide)
	}
	return names
}

// Version is set at build time via -ldflags.
var Version = "dev"

// UserAgent returns the user agent string for Roland.
func UserAgent() string {
	return fmt.Sprintf("roland/%s", Version)
}
