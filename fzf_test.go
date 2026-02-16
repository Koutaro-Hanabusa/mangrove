package mangrove

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestReorderWithDefault(t *testing.T) {
	tests := []struct {
		name        string
		items       []string
		defaultItem string
		want        []string
	}{
		{
			name:        "default exists in list",
			items:       []string{"feature", "main", "develop"},
			defaultItem: "main",
			want:        []string{"main", "feature", "develop"},
		},
		{
			name:        "default does not exist in list",
			items:       []string{"feature", "develop"},
			defaultItem: "main",
			want:        []string{"feature", "develop"},
		},
		{
			name:        "empty default",
			items:       []string{"feature", "main", "develop"},
			defaultItem: "",
			want:        []string{"feature", "main", "develop"},
		},
		{
			name:        "single item list with match",
			items:       []string{"main"},
			defaultItem: "main",
			want:        []string{"main"},
		},
		{
			name:        "single item list without match",
			items:       []string{"develop"},
			defaultItem: "main",
			want:        []string{"develop"},
		},
		{
			name:        "default already first stays first",
			items:       []string{"main", "develop", "feature"},
			defaultItem: "main",
			want:        []string{"main", "develop", "feature"},
		},
		{
			name:        "empty list returns empty",
			items:       []string{},
			defaultItem: "main",
			want:        []string{},
		},
		{
			name:        "default is last item",
			items:       []string{"a", "b", "c"},
			defaultItem: "c",
			want:        []string{"c", "a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reorderWithDefault(tt.items, tt.defaultItem)
			if len(got) != len(tt.want) {
				t.Fatalf("reorderWithDefault(%v, %q) returned %d items, want %d", tt.items, tt.defaultItem, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("reorderWithDefault(%v, %q)[%d] = %q, want %q", tt.items, tt.defaultItem, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestReorderWithDefaultDoesNotMutateInput(t *testing.T) {
	original := []string{"develop", "feature", "main"}
	copied := make([]string, len(original))
	copy(copied, original)

	_ = reorderWithDefault(original, "main")

	for i := range original {
		if original[i] != copied[i] {
			t.Errorf("reorderWithDefault mutated input slice: index %d changed from %q to %q", i, copied[i], original[i])
		}
	}
}

func TestFindGitRepositories(t *testing.T) {
	tmpDir := t.TempDir()

	gitRepos := []string{
		filepath.Join(tmpDir, "project-a"),
		filepath.Join(tmpDir, "workspace", "project-b"),
		filepath.Join(tmpDir, "workspace", "project-c"),
	}
	for _, repo := range gitRepos {
		gitDir := filepath.Join(repo, ".git")
		if err := os.MkdirAll(gitDir, 0o755); err != nil {
			t.Fatalf("failed to create test git dir %s: %v", gitDir, err)
		}
	}

	// Create non-git directories (should not appear in results)
	nonGitDirs := []string{
		filepath.Join(tmpDir, "plain-dir"),
		filepath.Join(tmpDir, "workspace", "docs"),
	}
	for _, dir := range nonGitDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create test dir %s: %v", dir, err)
		}
	}

	// Create a directory that should be skipped (node_modules)
	skippedRepo := filepath.Join(tmpDir, "node_modules", "some-package")
	if err := os.MkdirAll(filepath.Join(skippedRepo, ".git"), 0o755); err != nil {
		t.Fatalf("failed to create skipped repo dir: %v", err)
	}

	repos, err := FindGitRepositories(tmpDir)
	if err != nil {
		t.Fatalf("FindGitRepositories(%q) returned error: %v", tmpDir, err)
	}

	sort.Strings(repos)
	sort.Strings(gitRepos)

	if len(repos) != len(gitRepos) {
		t.Fatalf("FindGitRepositories returned %d repos, want %d\ngot:  %v\nwant: %v", len(repos), len(gitRepos), repos, gitRepos)
	}

	for i := range repos {
		if repos[i] != gitRepos[i] {
			t.Errorf("FindGitRepositories result[%d] = %q, want %q", i, repos[i], gitRepos[i])
		}
	}
}

func TestFindGitRepositoriesSkipsDirs(t *testing.T) {
	tmpDir := t.TempDir()

	skippedNames := []string{"node_modules", ".cache", ".npm", ".cargo", "vendor", "Library"}
	for _, name := range skippedNames {
		repoDir := filepath.Join(tmpDir, name, "hidden-repo", ".git")
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", repoDir, err)
		}
	}

	repos, err := FindGitRepositories(tmpDir)
	if err != nil {
		t.Fatalf("FindGitRepositories(%q) returned error: %v", tmpDir, err)
	}

	if len(repos) != 0 {
		t.Errorf("FindGitRepositories should skip directories in skipDirs, but found: %v", repos)
	}
}

func TestFindGitRepositoriesDoesNotDescendIntoRepo(t *testing.T) {
	tmpDir := t.TempDir()

	parentRepo := filepath.Join(tmpDir, "parent")
	nestedRepo := filepath.Join(parentRepo, "subdir", "nested")
	if err := os.MkdirAll(filepath.Join(parentRepo, ".git"), 0o755); err != nil {
		t.Fatalf("failed to create parent repo: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(nestedRepo, ".git"), 0o755); err != nil {
		t.Fatalf("failed to create nested repo: %v", err)
	}

	repos, err := FindGitRepositories(tmpDir)
	if err != nil {
		t.Fatalf("FindGitRepositories(%q) returned error: %v", tmpDir, err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo (parent only), got %d: %v", len(repos), repos)
	}
	if repos[0] != parentRepo {
		t.Errorf("expected %q, got %q", parentRepo, repos[0])
	}
}

func TestFindGitRepositoriesEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	repos, err := FindGitRepositories(tmpDir)
	if err != nil {
		t.Fatalf("FindGitRepositories(%q) returned error: %v", tmpDir, err)
	}

	if len(repos) != 0 {
		t.Errorf("expected 0 repos in empty dir, got %d: %v", len(repos), repos)
	}
}

func TestErrCancelled(t *testing.T) {
	if ErrCancelled == nil {
		t.Fatal("ErrCancelled should not be nil")
	}

	if ErrCancelled.Error() != "selection cancelled by user" {
		t.Errorf("ErrCancelled.Error() = %q, want %q", ErrCancelled.Error(), "selection cancelled by user")
	}

	wrapped := fmt.Errorf("some context: %w", ErrCancelled)
	if !errors.Is(wrapped, ErrCancelled) {
		t.Error("errors.Is(wrapped, ErrCancelled) should be true for wrapped error")
	}

	unrelated := fmt.Errorf("something else went wrong")
	if errors.Is(unrelated, ErrCancelled) {
		t.Error("errors.Is(unrelated, ErrCancelled) should be false for unrelated error")
	}
}

func TestFzfExitCodeHandling(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available, skipping exit code test")
	}

	tests := []struct {
		name     string
		exitCode int
		wantMsg  string
	}{
		{
			name:     "exit code 1 (ESC) treated as cancellation",
			exitCode: 1,
			wantMsg:  "cancelled",
		},
		{
			name:     "exit code 130 (Ctrl+C) treated as cancellation",
			exitCode: 130,
			wantMsg:  "cancelled",
		},
		{
			name:     "exit code 2 treated as error",
			exitCode: 2,
			wantMsg:  "failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath := filepath.Join(t.TempDir(), "mock-fzf.sh")
			script := fmt.Sprintf("#!/bin/bash\nexit %d\n", tt.exitCode)
			if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
				t.Fatalf("failed to write mock script: %v", err)
			}

			cmd := exec.Command("bash", scriptPath)
			_, err := cmd.Output()
			if err == nil {
				t.Fatal("expected error from non-zero exit code, got nil")
			}

			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("expected *exec.ExitError, got %T", err)
			}

			var resultErr error
			if exitErr.ExitCode() == 130 || exitErr.ExitCode() == 1 {
				resultErr = fmt.Errorf("%w", ErrCancelled)
			} else {
				resultErr = fmt.Errorf("fzf selection failed: %w", err)
			}

			if !strings.Contains(resultErr.Error(), tt.wantMsg) {
				t.Errorf("error message %q does not contain %q", resultErr.Error(), tt.wantMsg)
			}
		})
	}
}
