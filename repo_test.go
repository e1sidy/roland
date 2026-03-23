package roland

import (
	"testing"
)

func TestRepoNameFromURL(t *testing.T) {
	tt := []struct {
		url  string
		want string
	}{
		{"https://github.com/org/backend.git", "backend"},
		{"https://github.com/org/backend", "backend"},
		{"https://github.com/org/backend/", "backend"},
		{"git@github.com:org/frontend.git", "frontend"},
		{"git@github.com:org/frontend", "frontend"},
		{"https://github.com/org/my-repo.git", "my-repo"},
		{"backend", "backend"},
	}
	for _, tc := range tt {
		if got := repoNameFromURL(tc.url); got != tc.want {
			t.Errorf("repoNameFromURL(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestListRepos_Empty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Home = t.TempDir()
	repos := ListRepos(cfg)
	if len(repos) != 0 {
		t.Errorf("ListRepos() = %d repos, want 0", len(repos))
	}
}

func TestListRepos_Sorted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Home = t.TempDir()
	cfg.Repos["zrepo"] = &RepoConfig{URL: "https://github.com/org/z.git"}
	cfg.Repos["arepo"] = &RepoConfig{URL: "https://github.com/org/a.git"}

	repos := ListRepos(cfg)
	if len(repos) != 2 {
		t.Fatalf("ListRepos() = %d repos, want 2", len(repos))
	}
	if repos[0].Name != "arepo" {
		t.Errorf("first repo = %q, want %q", repos[0].Name, "arepo")
	}
	if repos[1].Name != "zrepo" {
		t.Errorf("second repo = %q, want %q", repos[1].Name, "zrepo")
	}
}

func TestAddRepo_DuplicateName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Home = t.TempDir()
	cfg.Repos["backend"] = &RepoConfig{URL: "https://github.com/org/backend.git"}

	_, err := AddRepo(cfg, "https://github.com/org/backend.git", "backend")
	if err == nil {
		t.Error("AddRepo should fail on duplicate name")
	}
}

func TestRemoveRepo_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Home = t.TempDir()

	err := RemoveRepo(cfg, "nonexistent", false)
	if err == nil {
		t.Error("RemoveRepo should fail for unknown repo")
	}
}

func TestSetPostSetup_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Home = t.TempDir()

	err := SetPostSetup(cfg, "nonexistent", "./setup.sh")
	if err == nil {
		t.Error("SetPostSetup should fail for unknown repo")
	}
}
