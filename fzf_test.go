package mangrove

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reorderWithDefault(tt.items, tt.defaultItem)
			if len(got) != len(tt.want) {
				t.Fatalf("reorderWithDefault() returned %d items, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("reorderWithDefault()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindGitRepositories(t *testing.T) {
	// Create a temp directory structure:
	// root/
	//   repo-a/.git/
	//   repo-b/.git/
	//   node_modules/hidden-repo/.git/  (should be skipped)
	//   nested/repo-c/.git/
	root := t.TempDir()

	dirs := []string{
		filepath.Join(root, "repo-a", ".git"),
		filepath.Join(root, "repo-b", ".git"),
		filepath.Join(root, "node_modules", "hidden-repo", ".git"),
		filepath.Join(root, "nested", "repo-c", ".git"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to create dir %s: %v", d, err)
		}
	}

	repos, err := FindGitRepositories(root)
	if err != nil {
		t.Fatalf("FindGitRepositories() error: %v", err)
	}

	// Build a set of found repos for easy lookup
	found := make(map[string]bool)
	for _, r := range repos {
		found[r] = true
	}

	// repo-a and repo-b should be found
	expectedRepos := []string{
		filepath.Join(root, "repo-a"),
		filepath.Join(root, "repo-b"),
		filepath.Join(root, "nested", "repo-c"),
	}
	for _, expected := range expectedRepos {
		if !found[expected] {
			t.Errorf("expected repo %q to be found, but it was not", expected)
		}
	}

	// node_modules should be skipped
	skippedRepo := filepath.Join(root, "node_modules", "hidden-repo")
	if found[skippedRepo] {
		t.Errorf("repo %q under node_modules should have been skipped", skippedRepo)
	}

	// Total count should match
	if len(repos) != len(expectedRepos) {
		t.Errorf("FindGitRepositories() found %d repos, want %d", len(repos), len(expectedRepos))
	}
}

func TestFindGitRepositories_EmptyDir(t *testing.T) {
	root := t.TempDir()

	repos, err := FindGitRepositories(root)
	if err != nil {
		t.Fatalf("FindGitRepositories() error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("FindGitRepositories() found %d repos in empty dir, want 0", len(repos))
	}
}

func TestErrCancelled(t *testing.T) {
	// Verify ErrCancelled is a proper error
	if ErrCancelled == nil {
		t.Fatal("ErrCancelled should not be nil")
	}

	if ErrCancelled.Error() != "selection cancelled by user" {
		t.Errorf("ErrCancelled.Error() = %q, want %q", ErrCancelled.Error(), "selection cancelled by user")
	}

	// Verify wrapped error can be unwrapped with errors.Is
	wrapped := fmt.Errorf("some context: %w", ErrCancelled)
	if !errors.Is(wrapped, ErrCancelled) {
		t.Error("errors.Is(wrapped, ErrCancelled) should be true for wrapped error")
	}

	// Verify double wrapping works
	doubleWrapped := fmt.Errorf("outer: %w", wrapped)
	if !errors.Is(doubleWrapped, ErrCancelled) {
		t.Error("errors.Is(doubleWrapped, ErrCancelled) should be true for double-wrapped error")
	}

	// Verify unrelated error does not match
	unrelated := fmt.Errorf("something else went wrong")
	if errors.Is(unrelated, ErrCancelled) {
		t.Error("errors.Is(unrelated, ErrCancelled) should be false for unrelated error")
	}
}
