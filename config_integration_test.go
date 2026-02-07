package mangrove

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestSaveAndLoadConfig(t *testing.T) {
	viper.Reset()

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := Config{
		BaseDir:        filepath.Join(tmpDir, "workspaces"),
		DefaultProfile: "test",
		Profiles: map[string]Profile{
			"test": {
				Repos: []Repo{
					{
						Name:        "app",
						Path:        filepath.Join(tmpDir, "repos", "app"),
						DefaultBase: "main",
					},
					{
						Name:        "lib",
						Path:        filepath.Join(tmpDir, "repos", "lib"),
						DefaultBase: "develop",
					},
				},
				Hooks: Hooks{
					PostCreate: []Hook{
						{Repo: "app", Run: "make setup"},
					},
				},
			},
		},
	}

	if err := SaveConfig(&cfg); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	// Verify the config file was created
	configPath := filepath.Join(tmpDir, ".config", "mgv", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not created at %s: %v", configPath, err)
	}

	// Reset viper before loading to clear global state
	viper.Reset()

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}

	// BaseDir should be expanded back to the absolute path
	if loaded.BaseDir != cfg.BaseDir {
		t.Errorf("BaseDir = %q, want %q", loaded.BaseDir, cfg.BaseDir)
	}

	if loaded.DefaultProfile != cfg.DefaultProfile {
		t.Errorf("DefaultProfile = %q, want %q", loaded.DefaultProfile, cfg.DefaultProfile)
	}

	if len(loaded.Profiles) != len(cfg.Profiles) {
		t.Fatalf("Profiles count = %d, want %d", len(loaded.Profiles), len(cfg.Profiles))
	}

	testProfile, ok := loaded.Profiles["test"]
	if !ok {
		t.Fatal("profile 'test' not found in loaded config")
	}

	if len(testProfile.Repos) != 2 {
		t.Fatalf("Repos count = %d, want 2", len(testProfile.Repos))
	}

	// Verify repos have expanded paths
	for i, repo := range testProfile.Repos {
		wantRepo := cfg.Profiles["test"].Repos[i]
		if repo.Name != wantRepo.Name {
			t.Errorf("Repo[%d].Name = %q, want %q", i, repo.Name, wantRepo.Name)
		}
		if repo.Path != wantRepo.Path {
			t.Errorf("Repo[%d].Path = %q, want %q", i, repo.Path, wantRepo.Path)
		}
		if repo.DefaultBase != wantRepo.DefaultBase {
			t.Errorf("Repo[%d].DefaultBase = %q, want %q", i, repo.DefaultBase, wantRepo.DefaultBase)
		}
	}

	// Verify hooks survived round-trip
	if len(testProfile.Hooks.PostCreate) != 1 {
		t.Fatalf("PostCreate hooks count = %d, want 1", len(testProfile.Hooks.PostCreate))
	}
	hook := testProfile.Hooks.PostCreate[0]
	if hook.Repo != "app" || hook.Run != "make setup" {
		t.Errorf("Hook = {Repo:%q, Run:%q}, want {Repo:\"app\", Run:\"make setup\"}", hook.Repo, hook.Run)
	}
}

func TestDetectDefaultBranch(t *testing.T) {
	t.Run("gitリポジトリなしはmainにフォールバック", func(t *testing.T) {
		tmpDir := t.TempDir()
		got := DetectDefaultBranch(tmpDir)
		if got != "main" {
			t.Errorf("DetectDefaultBranch(non-repo) = %q, want \"main\"", got)
		}
	})

	t.Run("リモート未設定のリポジトリはmainにフォールバック", func(t *testing.T) {
		tmpDir := t.TempDir()

		cmd := exec.Command("git", "init", tmpDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %s: %v", out, err)
		}

		got := DetectDefaultBranch(tmpDir)
		if got != "main" {
			t.Errorf("DetectDefaultBranch(no-remote) = %q, want \"main\"", got)
		}
	})
}
