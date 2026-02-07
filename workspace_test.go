package mangrove

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetWorkspacePath(t *testing.T) {
	tests := []struct {
		name        string
		baseDir     string
		profileName string
		wsName      string
		want        string
	}{
		{
			name:        "標準的なパス組み立て",
			baseDir:     "/base",
			profileName: "dev",
			wsName:      "feature",
			want:        filepath.Join("/base", "dev", "feature"),
		},
		{
			name:        "プロファイル名が空の場合",
			baseDir:     "/base",
			profileName: "",
			wsName:      "ws",
			want:        filepath.Join("/base", "ws"),
		},
		{
			name:        "ワークスペース名が空の場合",
			baseDir:     "/base",
			profileName: "p",
			wsName:      "",
			want:        filepath.Join("/base", "p"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{BaseDir: tt.baseDir}
			got := GetWorkspacePath(cfg, tt.profileName, tt.wsName)
			if got != tt.want {
				t.Errorf("GetWorkspacePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWorkspaceLabels(t *testing.T) {
	t.Run("クリーンなリポジトリのラベル生成", func(t *testing.T) {
		workspaces := []WorkspaceInfo{
			{
				ProfileName:   "dev",
				WorkspaceName: "feature-1",
				Path:          "/base/dev/feature-1",
				RepoStatuses: []RepoStatus{
					{RepoName: "api", ChangedCount: 0, Exists: true},
				},
			},
		}

		labels := WorkspaceLabels(workspaces)
		if len(labels) != 1 {
			t.Fatalf("expected 1 label, got %d", len(labels))
		}
		if !strings.HasPrefix(labels[0], "dev/feature-1") {
			t.Errorf("label should start with %q, got %q", "dev/feature-1", labels[0])
		}
		if !strings.Contains(labels[0], "api") {
			t.Errorf("label should contain repo name %q, got %q", "api", labels[0])
		}
	})

	t.Run("変更ありリポジトリのラベル生成", func(t *testing.T) {
		workspaces := []WorkspaceInfo{
			{
				ProfileName:   "dev",
				WorkspaceName: "feature-2",
				Path:          "/base/dev/feature-2",
				RepoStatuses: []RepoStatus{
					{RepoName: "web", ChangedCount: 3, Exists: true},
				},
			},
		}

		labels := WorkspaceLabels(workspaces)
		if len(labels) != 1 {
			t.Fatalf("expected 1 label, got %d", len(labels))
		}
		if !strings.HasPrefix(labels[0], "dev/feature-2") {
			t.Errorf("label should start with %q, got %q", "dev/feature-2", labels[0])
		}
		if !strings.Contains(labels[0], "web") {
			t.Errorf("label should contain repo name %q, got %q", "web", labels[0])
		}
	})

	t.Run("存在しないリポジトリはmissing表示", func(t *testing.T) {
		workspaces := []WorkspaceInfo{
			{
				ProfileName:   "prod",
				WorkspaceName: "hotfix",
				Path:          "/base/prod/hotfix",
				RepoStatuses: []RepoStatus{
					{RepoName: "backend", Exists: false},
				},
			},
		}

		labels := WorkspaceLabels(workspaces)
		if len(labels) != 1 {
			t.Fatalf("expected 1 label, got %d", len(labels))
		}
		if !strings.HasPrefix(labels[0], "prod/hotfix") {
			t.Errorf("label should start with %q, got %q", "prod/hotfix", labels[0])
		}
		if !strings.Contains(labels[0], "missing") {
			t.Errorf("label should contain %q for non-existing repo, got %q", "missing", labels[0])
		}
	})

	t.Run("空のワークスペースリストは空ラベル", func(t *testing.T) {
		labels := WorkspaceLabels([]WorkspaceInfo{})
		if len(labels) != 0 {
			t.Errorf("expected 0 labels, got %d", len(labels))
		}
	})
}

func TestParseWorkspaceLabel(t *testing.T) {
	tests := []struct {
		name          string
		label         string
		wantProfile   string
		wantWorkspace string
		wantErr       bool
	}{
		{
			name:          "ステータス付きラベルのパース",
			label:         "dev/feature-1     [repo: \u2713 clean]",
			wantProfile:   "dev",
			wantWorkspace: "feature-1",
			wantErr:       false,
		},
		{
			name:          "ステータスなしラベルのパース",
			label:         "prod/hotfix",
			wantProfile:   "prod",
			wantWorkspace: "hotfix",
			wantErr:       false,
		},
		{
			name:    "スラッシュなしはエラー",
			label:   "noslash",
			wantErr: true,
		},
		{
			name:    "空ラベルはエラー",
			label:   "",
			wantErr: true,
		},
		{
			name:          "複数スラッシュはSplitNで2分割",
			label:         "a/b/c",
			wantProfile:   "a",
			wantWorkspace: "b/c",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, workspace, err := ParseWorkspaceLabel(tt.label)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseWorkspaceLabel(%q) expected error, got nil", tt.label)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseWorkspaceLabel(%q) unexpected error: %v", tt.label, err)
				return
			}
			if profile != tt.wantProfile {
				t.Errorf("ParseWorkspaceLabel(%q) profile = %q, want %q", tt.label, profile, tt.wantProfile)
			}
			if workspace != tt.wantWorkspace {
				t.Errorf("ParseWorkspaceLabel(%q) workspace = %q, want %q", tt.label, workspace, tt.wantWorkspace)
			}
		})
	}
}
