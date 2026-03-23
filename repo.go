package roland

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/e1sidy/roland/internal/gitutil"
)

// Repo represents a registered codebase.
type Repo struct {
	Name string
	URL  string
	Path string
}

// SyncResult holds the result of syncing a repo.
type SyncResult struct {
	Name    string
	Updated bool
	Error   error
}

// AddRepo clones a repo and registers it in the config.
// If name is empty, it is derived from the URL.
func AddRepo(cfg *Config, url, name string) (*Repo, error) {
	if name == "" {
		name = repoNameFromURL(url)
	}
	if name == "" {
		return nil, fmt.Errorf("cannot derive repo name from URL %q", url)
	}

	if _, exists := cfg.Repos[name]; exists {
		return nil, fmt.Errorf("repo %q already registered", name)
	}

	repoDir := filepath.Join(ReposDir(cfg.Home), name)
	if err := os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return nil, fmt.Errorf("create repos dir: %w", err)
	}

	if err := gitutil.Run(cfg.Home, "clone", url, repoDir); err != nil {
		return nil, fmt.Errorf("clone %s: %w", url, err)
	}

	rc := &RepoConfig{
		URL:        url,
		BaseBranch: "origin/main",
	}
	cfg.Repos[name] = rc

	if err := SaveConfig(cfg); err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	return &Repo{Name: name, URL: url, Path: repoDir}, nil
}

// RemoveRepo unregisters a repo and optionally deletes its files.
func RemoveRepo(cfg *Config, name string, deleteFiles bool) error {
	if _, exists := cfg.Repos[name]; !exists {
		return fmt.Errorf("repo %q not registered", name)
	}

	delete(cfg.Repos, name)
	if err := SaveConfig(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	if deleteFiles {
		repoDir := filepath.Join(ReposDir(cfg.Home), name)
		if err := os.RemoveAll(repoDir); err != nil {
			return fmt.Errorf("remove repo dir: %w", err)
		}
	}

	return nil
}

// ListRepos returns all registered repos, sorted by name.
func ListRepos(cfg *Config) []*Repo {
	repos := make([]*Repo, 0, len(cfg.Repos))
	for name, rc := range cfg.Repos {
		repos = append(repos, &Repo{
			Name: name,
			URL:  rc.URL,
			Path: filepath.Join(ReposDir(cfg.Home), name),
		})
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})
	return repos
}

// SyncRepo fetches and fast-forwards a registered repo.
func SyncRepo(cfg *Config, name string) (*SyncResult, error) {
	if _, exists := cfg.Repos[name]; !exists {
		return nil, fmt.Errorf("repo %q not registered", name)
	}

	repoDir := filepath.Join(ReposDir(cfg.Home), name)
	result := &SyncResult{Name: name}

	if err := gitutil.Run(repoDir, "fetch", "--prune"); err != nil {
		result.Error = fmt.Errorf("fetch %s: %w", name, err)
		return result, result.Error
	}

	// Try fast-forward merge on current branch.
	out, err := gitutil.Output(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return result, nil // Detached HEAD — skip merge.
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return result, nil
	}

	if err := gitutil.Run(repoDir, "merge", "--ff-only", "origin/"+branch); err != nil {
		// Not fast-forwardable — that's okay.
		return result, nil
	}
	result.Updated = true
	return result, nil
}

// SyncAllRepos syncs every registered repo.
func SyncAllRepos(cfg *Config) []*SyncResult {
	results := make([]*SyncResult, 0, len(cfg.Repos))
	for name := range cfg.Repos {
		r, _ := SyncRepo(cfg, name)
		if r != nil {
			results = append(results, r)
		}
	}
	return results
}

// SetPostSetup sets the post-setup script for a repo.
func SetPostSetup(cfg *Config, repoName, scriptPath string) error {
	rc, exists := cfg.Repos[repoName]
	if !exists {
		return fmt.Errorf("repo %q not registered", repoName)
	}
	rc.PostSetup = scriptPath
	return SaveConfig(cfg)
}

// repoNameFromURL extracts a short name from a git URL.
// Examples:
//
//	https://github.com/org/repo.git → repo
//	git@github.com:org/repo.git    → repo
//	https://github.com/org/repo/   → repo
func repoNameFromURL(url string) string {
	// Remove trailing slash and .git suffix.
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")

	// Get last path component.
	if idx := strings.LastIndex(url, "/"); idx >= 0 {
		return url[idx+1:]
	}
	// Handle SSH colon separator (git@github.com:org/repo).
	if idx := strings.LastIndex(url, ":"); idx >= 0 {
		rest := url[idx+1:]
		if slash := strings.LastIndex(rest, "/"); slash >= 0 {
			return rest[slash+1:]
		}
		return rest
	}
	return url
}
