package main

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/e1sidy/roland"
)

// launchAgent replaces the current process with the given AI coding agent.
// The agent is launched in the specified directory with any extra flags.
func launchAgent(dir string, agent roland.AgentTool, flags []string) error {
	binary, err := exec.LookPath(agent.Command())
	if err != nil {
		return fmt.Errorf("agent %q not found in PATH: %w", agent.Command(), err)
	}

	// Build argv: [command, flags..., dir]
	argv := []string{agent.Command()}
	argv = append(argv, flags...)

	// For claude, pass the directory as a positional arg via --dir.
	// For opencode, pass it as the working directory.
	switch agent {
	case roland.AgentClaude:
		argv = append(argv, "--dir", dir)
	case roland.AgentCodex:
		argv = append(argv, "--cwd", dir)
	case roland.AgentGemini:
		argv = append(argv, "--project-dir", dir)
	default:
		// Generic: just use the directory as cwd (set via env).
	}

	// Replace the current process.
	return syscall.Exec(binary, argv, syscall.Environ())
}
