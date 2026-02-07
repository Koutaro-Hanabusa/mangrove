package mangrove

import (
	"os"
	"sort"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"チルダ付きサブディレクトリを展開", "~/Documents", home + "/Documents"},
		{"絶対パスはそのまま", "/absolute/path", "/absolute/path"},
		{"相対パスはそのまま", "relative/path", "relative/path"},
		{"空文字列はそのまま", "", ""},
		{"チルダの後にスラッシュなしはそのまま", "~notslash", "~notslash"},
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
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"ホーム配下のパスをチルダ形式に変換", home + "/Documents", "~/Documents"},
		{"ホーム外のパスはそのまま", "/other/path", "/other/path"},
		{"ホームディレクトリ自体はチルダに変換", home, "~"},
		{"空文字列はそのまま", "", ""},
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
		DefaultProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Repos: []Repo{{Name: "repo1", Path: "/path/to/repo1"}},
			},
			"work": {
				Repos: []Repo{{Name: "repo2", Path: "/path/to/repo2"}},
			},
		},
	}

	tests := []struct {
		name        string
		profileName string
		wantName    string
		wantErr     bool
	}{
		{"名前指定で既存プロファイルを取得", "work", "work", false},
		{"存在しないプロファイルはエラー", "missing", "", true},
		{"空名はデフォルトプロファイルを使用", "", "default", false},
		{"デフォルト未設定で空名はエラー", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cfg
			if tt.name == "デフォルト未設定で空名はエラー" {
				c = &Config{
					DefaultProfile: "",
					Profiles:       map[string]Profile{},
				}
			}

			profile, name, err := c.GetProfile(tt.profileName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetProfile(%q) expected error, got nil", tt.profileName)
				}
				return
			}
			if err != nil {
				t.Errorf("GetProfile(%q) unexpected error: %v", tt.profileName, err)
				return
			}
			if name != tt.wantName {
				t.Errorf("GetProfile(%q) name = %q, want %q", tt.profileName, name, tt.wantName)
			}
			if profile == nil {
				t.Errorf("GetProfile(%q) returned nil profile", tt.profileName)
			}
		})
	}
}

func TestProfileNames(t *testing.T) {
	tests := []struct {
		name     string
		profiles map[string]Profile
		want     []string
	}{
		{
			"複数プロファイルの名前一覧を取得",
			map[string]Profile{
				"alpha": {},
				"beta":  {},
				"gamma": {},
			},
			[]string{"alpha", "beta", "gamma"},
		},
		{
			"プロファイルなしは空スライス",
			map[string]Profile{},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Profiles: tt.profiles}
			got := cfg.ProfileNames()
			sort.Strings(got)
			sort.Strings(tt.want)
			if len(got) != len(tt.want) {
				t.Errorf("ProfileNames() returned %d names, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ProfileNames()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetDefaultBase(t *testing.T) {
	tests := []struct {
		name        string
		defaultBase string
		want        string
	}{
		{"カスタムベースブランチを返す", "develop", "develop"},
		{"未設定時はmainにフォールバック", "", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Repo{DefaultBase: tt.defaultBase}
			got := r.GetDefaultBase()
			if got != tt.want {
				t.Errorf("GetDefaultBase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddProfile(t *testing.T) {
	tests := []struct {
		name    string
		setup   map[string]Profile
		addName string
		wantErr bool
	}{
		{"新しいプロファイルを正常に追加", map[string]Profile{}, "new-profile", false},
		{"既存プロファイル名はエラー", map[string]Profile{"existing": {}}, "existing", true},
		{"nilマップでも正常に追加", nil, "first", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Profiles: tt.setup}
			err := cfg.AddProfile(tt.addName, Profile{})
			if tt.wantErr {
				if err == nil {
					t.Errorf("AddProfile(%q) expected error, got nil", tt.addName)
				}
				return
			}
			if err != nil {
				t.Errorf("AddProfile(%q) unexpected error: %v", tt.addName, err)
				return
			}
			if _, ok := cfg.Profiles[tt.addName]; !ok {
				t.Errorf("AddProfile(%q) profile not found after adding", tt.addName)
			}
		})
	}
}

func TestAddRepoToProfile(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		repo        Repo
		wantErr     bool
	}{
		{"リポジトリを正常に追加", "dev", Repo{Name: "new-repo", Path: "/path"}, false},
		{"存在しないプロファイルはエラー", "nonexistent", Repo{Name: "repo"}, true},
		{"同名リポジトリの重複はエラー", "dev", Repo{Name: "existing-repo"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Profiles: map[string]Profile{
					"dev": {
						Repos: []Repo{{Name: "existing-repo", Path: "/existing"}},
					},
				},
			}
			err := cfg.AddRepoToProfile(tt.profileName, tt.repo)
			if tt.wantErr {
				if err == nil {
					t.Errorf("AddRepoToProfile(%q, %q) expected error, got nil", tt.profileName, tt.repo.Name)
				}
				return
			}
			if err != nil {
				t.Errorf("AddRepoToProfile(%q, %q) unexpected error: %v", tt.profileName, tt.repo.Name, err)
				return
			}
			profile := cfg.Profiles[tt.profileName]
			found := false
			for _, r := range profile.Repos {
				if r.Name == tt.repo.Name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("AddRepoToProfile(%q, %q) repo not found after adding", tt.profileName, tt.repo.Name)
			}
		})
	}
}

func TestRemoveRepoFromProfile(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		repoName    string
		wantErr     bool
	}{
		{"リポジトリを正常に削除", "dev", "repo1", false},
		{"存在しないプロファイルはエラー", "nonexistent", "repo1", true},
		{"存在しないリポジトリはエラー", "dev", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Profiles: map[string]Profile{
					"dev": {
						Repos: []Repo{
							{Name: "repo1", Path: "/path/repo1"},
							{Name: "repo2", Path: "/path/repo2"},
						},
					},
				},
			}
			err := cfg.RemoveRepoFromProfile(tt.profileName, tt.repoName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveRepoFromProfile(%q, %q) expected error, got nil", tt.profileName, tt.repoName)
				}
				return
			}
			if err != nil {
				t.Errorf("RemoveRepoFromProfile(%q, %q) unexpected error: %v", tt.profileName, tt.repoName, err)
				return
			}
			profile := cfg.Profiles[tt.profileName]
			if len(profile.Repos) != 1 {
				t.Errorf("RemoveRepoFromProfile(%q, %q) expected 1 repo remaining, got %d", tt.profileName, tt.repoName, len(profile.Repos))
			}
			for _, r := range profile.Repos {
				if r.Name == tt.repoName {
					t.Errorf("RemoveRepoFromProfile(%q, %q) repo still present after removal", tt.profileName, tt.repoName)
				}
			}
		})
	}
}
