package mangrove

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "tilde with subpath",
			path: "~/Documents/projects",
			want: filepath.Join(home, "Documents", "projects"),
		},
		{
			name: "non-tilde path passthrough",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "exact tilde only is not expanded",
			path: "~",
			want: "~",
		},
		{
			name: "relative path passthrough",
			path: "relative/path",
			want: "relative/path",
		},
		{
			name: "tilde slash root",
			path: "~/",
			want: home,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.path)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCollapsePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "home dir with subpath",
			path: filepath.Join(home, "Documents", "projects"),
			want: "~/Documents/projects",
		},
		{
			name: "non-home path passthrough",
			path: "/usr/local/bin",
			want: "/usr/local/bin",
		},
		{
			name: "exact home dir",
			path: home,
			want: "~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollapsePath(tt.path)
			if got != tt.want {
				t.Errorf("CollapsePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	cfg := &Config{
		DefaultProfile: "default-profile",
		Profiles: map[string]Profile{
			"default-profile": {
				Repos: []Repo{{Name: "repo1", Path: "/path/to/repo1"}},
			},
			"other-profile": {
				Repos: []Repo{{Name: "repo2", Path: "/path/to/repo2"}},
			},
		},
	}

	tests := []struct {
		name      string
		input     string
		wantName  string
		wantRepos int
		wantErr   bool
	}{
		{
			name:      "existing profile by name",
			input:     "other-profile",
			wantName:  "other-profile",
			wantRepos: 1,
			wantErr:   false,
		},
		{
			name:    "missing profile",
			input:   "nonexistent",
			wantErr: true,
		},
		{
			name:      "empty name uses default",
			input:     "",
			wantName:  "default-profile",
			wantRepos: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, name, err := cfg.GetProfile(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetProfile(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetProfile(%q) unexpected error: %v", tt.input, err)
			}
			if name != tt.wantName {
				t.Errorf("GetProfile(%q) name = %q, want %q", tt.input, name, tt.wantName)
			}
			if len(profile.Repos) != tt.wantRepos {
				t.Errorf("GetProfile(%q) repos = %d, want %d", tt.input, len(profile.Repos), tt.wantRepos)
			}
		})
	}

	// Test empty name without default
	cfgNoDefault := &Config{
		DefaultProfile: "",
		Profiles: map[string]Profile{
			"some-profile": {},
		},
	}
	_, _, err2 := cfgNoDefault.GetProfile("")
	if err2 == nil {
		t.Error("GetProfile(\"\") with no default should return error")
	}
}

func TestProfileNames(t *testing.T) {
	tests := []struct {
		name     string
		profiles map[string]Profile
		wantLen  int
	}{
		{
			name: "with profiles",
			profiles: map[string]Profile{
				"alpha": {},
				"beta":  {},
				"gamma": {},
			},
			wantLen: 3,
		},
		{
			name:     "empty profiles",
			profiles: map[string]Profile{},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Profiles: tt.profiles}
			names := cfg.ProfileNames()
			if len(names) != tt.wantLen {
				t.Errorf("ProfileNames() returned %d names, want %d", len(names), tt.wantLen)
			}

			// Verify all profile names are present
			sort.Strings(names)
			expectedNames := make([]string, 0, len(tt.profiles))
			for n := range tt.profiles {
				expectedNames = append(expectedNames, n)
			}
			sort.Strings(expectedNames)

			for i, name := range names {
				if name != expectedNames[i] {
					t.Errorf("ProfileNames()[%d] = %q, want %q", i, name, expectedNames[i])
				}
			}
		})
	}
}

func TestAddProfile(t *testing.T) {
	t.Run("normal add", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{},
		}
		profile := Profile{Repos: []Repo{{Name: "repo1", Path: "/path"}}}
		err := cfg.AddProfile("new-profile", profile)
		if err != nil {
			t.Fatalf("AddProfile() unexpected error: %v", err)
		}
		if _, ok := cfg.Profiles["new-profile"]; !ok {
			t.Error("AddProfile() profile not added to map")
		}
	})

	t.Run("duplicate add error", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{
				"existing": {},
			},
		}
		err := cfg.AddProfile("existing", Profile{})
		if err == nil {
			t.Error("AddProfile() expected error for duplicate profile")
		}
	})

	t.Run("nil map initialization", func(t *testing.T) {
		cfg := &Config{
			Profiles: nil,
		}
		err := cfg.AddProfile("new-profile", Profile{})
		if err != nil {
			t.Fatalf("AddProfile() unexpected error: %v", err)
		}
		if cfg.Profiles == nil {
			t.Error("AddProfile() should initialize nil Profiles map")
		}
		if _, ok := cfg.Profiles["new-profile"]; !ok {
			t.Error("AddProfile() profile not added after nil map init")
		}
	})
}

func TestAddRepoToProfile(t *testing.T) {
	t.Run("normal add", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{
				"myprofile": {Repos: []Repo{{Name: "repo1", Path: "/path/repo1"}}},
			},
		}
		repo := Repo{Name: "repo2", Path: "/path/repo2"}
		err := cfg.AddRepoToProfile("myprofile", repo)
		if err != nil {
			t.Fatalf("AddRepoToProfile() unexpected error: %v", err)
		}
		profile := cfg.Profiles["myprofile"]
		if len(profile.Repos) != 2 {
			t.Errorf("AddRepoToProfile() repos count = %d, want 2", len(profile.Repos))
		}
	})

	t.Run("duplicate repo error", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{
				"myprofile": {Repos: []Repo{{Name: "repo1", Path: "/path/repo1"}}},
			},
		}
		repo := Repo{Name: "repo1", Path: "/path/repo1-dup"}
		err := cfg.AddRepoToProfile("myprofile", repo)
		if err == nil {
			t.Error("AddRepoToProfile() expected error for duplicate repo name")
		}
	})

	t.Run("missing profile error", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{},
		}
		repo := Repo{Name: "repo1", Path: "/path/repo1"}
		err := cfg.AddRepoToProfile("nonexistent", repo)
		if err == nil {
			t.Error("AddRepoToProfile() expected error for missing profile")
		}
	})
}

func TestRemoveRepoFromProfile(t *testing.T) {
	t.Run("normal remove", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{
				"myprofile": {
					Repos: []Repo{
						{Name: "repo1", Path: "/path/repo1"},
						{Name: "repo2", Path: "/path/repo2"},
					},
				},
			},
		}
		err := cfg.RemoveRepoFromProfile("myprofile", "repo1")
		if err != nil {
			t.Fatalf("RemoveRepoFromProfile() unexpected error: %v", err)
		}
		profile := cfg.Profiles["myprofile"]
		if len(profile.Repos) != 1 {
			t.Errorf("RemoveRepoFromProfile() repos count = %d, want 1", len(profile.Repos))
		}
		if profile.Repos[0].Name != "repo2" {
			t.Errorf("RemoveRepoFromProfile() remaining repo = %q, want %q", profile.Repos[0].Name, "repo2")
		}
	})

	t.Run("missing repo error", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{
				"myprofile": {Repos: []Repo{{Name: "repo1", Path: "/path/repo1"}}},
			},
		}
		err := cfg.RemoveRepoFromProfile("myprofile", "nonexistent")
		if err == nil {
			t.Error("RemoveRepoFromProfile() expected error for missing repo")
		}
	})

	t.Run("missing profile error", func(t *testing.T) {
		cfg := &Config{
			Profiles: map[string]Profile{},
		}
		err := cfg.RemoveRepoFromProfile("nonexistent", "repo1")
		if err == nil {
			t.Error("RemoveRepoFromProfile() expected error for missing profile")
		}
	})
}

func TestGetDefaultBase(t *testing.T) {
	tests := []struct {
		name        string
		defaultBase string
		want        string
	}{
		{
			name:        "with value set",
			defaultBase: "develop",
			want:        "develop",
		},
		{
			name:        "empty falls back to main",
			defaultBase: "",
			want:        "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repo{DefaultBase: tt.defaultBase}
			got := repo.GetDefaultBase()
			if got != tt.want {
				t.Errorf("GetDefaultBase() = %q, want %q", got, tt.want)
			}
		})
	}
}
