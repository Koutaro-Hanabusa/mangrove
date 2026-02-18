package mangrove

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initTestRepo はテスト用のgitリポジトリを作成する。
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init", "-b", "main")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "initial commit")
	return dir
}

// gitRun はテスト内でgitコマンドを実行する。
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed in %s: %s: %v", args, dir, output, err)
	}
	return strings.TrimSpace(string(output))
}

func TestStashPushAndPop(t *testing.T) {
	repo := initTestRepo(t)

	// ファイルを変更
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# changed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// stash push
	if err := StashPush(repo, "test stash"); err != nil {
		t.Fatalf("StashPush failed: %v", err)
	}

	// ワーキングツリーがクリーンになったことを確認
	status, err := StatusPorcelain(repo)
	if err != nil {
		t.Fatal(err)
	}
	if status != "" {
		t.Errorf("stash push後もワーキングツリーが汚れている: %q", status)
	}

	// stash pop
	if err := StashPop(repo); err != nil {
		t.Fatalf("StashPop failed: %v", err)
	}

	// 変更が復元されたことを確認
	status, err = StatusPorcelain(repo)
	if err != nil {
		t.Fatal(err)
	}
	if status == "" {
		t.Error("stash pop後に変更が復元されていない")
	}
}

func TestStashPushNoChanges(t *testing.T) {
	repo := initTestRepo(t)

	// 変更なしでstash pushしてもgitはexit 0を返す（エラーにはならない）
	err := StashPush(repo, "no changes")
	if err != nil {
		t.Errorf("変更なしのStashPushが予期せずエラー: %v", err)
	}

	// stashリストが空であることを確認（何もstashされていない）
	output := gitRun(t, repo, "stash", "list")
	if output != "" {
		t.Errorf("変更なしのStashPush後にstashが作成されている: %q", output)
	}
}

func TestStashRefAndApply(t *testing.T) {
	repo := initTestRepo(t)

	// ファイルを変更してstash
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := StashPush(repo, "test ref"); err != nil {
		t.Fatal(err)
	}

	// stash SHA を取得
	ref, err := StashRef(repo)
	if err != nil {
		t.Fatalf("StashRef failed: %v", err)
	}
	if ref == "" {
		t.Fatal("StashRef が空のSHAを返した")
	}

	// stash drop してreflogから消す
	if err := StashDrop(repo); err != nil {
		t.Fatalf("StashDrop failed: %v", err)
	}

	// SHA経由でstash apply（reflogになくてもオブジェクトは残っている）
	if err := StashApply(repo, ref); err != nil {
		t.Fatalf("StashApply failed: %v", err)
	}

	// 変更が適用されたことを確認
	content, err := os.ReadFile(filepath.Join(repo, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# changed\n" {
		t.Errorf("StashApply後のファイル内容 = %q, want %q", string(content), "# changed\n")
	}
}

func TestStashDropEmpty(t *testing.T) {
	repo := initTestRepo(t)

	// stashが空の状態でdropするとエラー
	err := StashDrop(repo)
	if err == nil {
		t.Error("stashが空の状態でStashDropがエラーにならなかった")
	}
}

func TestMergeAbort(t *testing.T) {
	repo := initTestRepo(t)

	// コンフリクトを発生させる
	gitRun(t, repo, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# feature\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "feature")

	gitRun(t, repo, "checkout", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "main")

	// マージ（コンフリクト）
	_ = Merge(repo, "feature")

	// MergeAbortが成功すること
	if err := MergeAbort(repo); err != nil {
		t.Fatalf("MergeAbort failed: %v", err)
	}

	// ワーキングツリーがクリーンに戻ること
	status, err := StatusPorcelain(repo)
	if err != nil {
		t.Fatal(err)
	}
	if status != "" {
		t.Errorf("MergeAbort後にワーキングツリーが汚れている: %q", status)
	}
}

func TestCheckoutBranch(t *testing.T) {
	repo := initTestRepo(t)

	// ブランチ作成
	gitRun(t, repo, "branch", "feature")

	// 切り替え
	if err := CheckoutBranch(repo, "feature"); err != nil {
		t.Fatalf("CheckoutBranch failed: %v", err)
	}

	branch, err := CurrentBranch(repo)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature" {
		t.Errorf("CheckoutBranch後のブランチ = %q, want %q", branch, "feature")
	}
}

func TestCheckoutBranchNotFound(t *testing.T) {
	repo := initTestRepo(t)

	err := CheckoutBranch(repo, "nonexistent")
	if err == nil {
		t.Error("存在しないブランチへのCheckoutBranchがエラーにならなかった")
	}
}

func TestCheckoutNewBranch(t *testing.T) {
	repo := initTestRepo(t)

	if err := CheckoutNewBranch(repo, "feature/new", "main"); err != nil {
		t.Fatalf("CheckoutNewBranch failed: %v", err)
	}

	branch, err := CurrentBranch(repo)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature/new" {
		t.Errorf("CheckoutNewBranch後のブランチ = %q, want %q", branch, "feature/new")
	}
}

func TestCheckoutNewBranchDuplicate(t *testing.T) {
	repo := initTestRepo(t)

	// 同名ブランチが既にあるとエラー
	err := CheckoutNewBranch(repo, "main", "main")
	if err == nil {
		t.Error("既存ブランチ名でCheckoutNewBranchがエラーにならなかった")
	}
}

func TestMerge(t *testing.T) {
	repo := initTestRepo(t)

	// featureブランチでコミット追加
	gitRun(t, repo, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "add feature")

	// mainに戻ってマージ
	gitRun(t, repo, "checkout", "main")
	if err := Merge(repo, "feature"); err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// マージ後にfeature.txtが存在することを確認
	if _, err := os.Stat(filepath.Join(repo, "feature.txt")); os.IsNotExist(err) {
		t.Error("マージ後にfeature.txtが存在しない")
	}
}

func TestMergeConflict(t *testing.T) {
	repo := initTestRepo(t)

	// featureブランチでREADME変更
	gitRun(t, repo, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# feature\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "feature change")

	// mainでも同じファイルを変更（コンフリクト発生）
	gitRun(t, repo, "checkout", "main")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("# main change\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "main change")

	err := Merge(repo, "feature")
	if err == nil {
		t.Error("コンフリクト時にMergeがエラーにならなかった")
	}

	// MergeAbortでクリーンアップ
	if err := MergeAbort(repo); err != nil {
		t.Fatalf("MergeAbort failed: %v", err)
	}
}

func TestParseLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "normal multiline input",
			input: "main\ndevelop\nfeature/foo\n",
			want:  []string{"main", "develop", "feature/foo"},
		},
		{
			name:  "empty lines filtered",
			input: "main\n\ndevelop\n\n",
			want:  []string{"main", "develop"},
		},
		{
			name:  "single line",
			input: "main\n",
			want:  []string{"main"},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "only whitespace lines",
			input: "\n  \n\t\n",
			want:  nil,
		},
		{
			name:  "lines with extra whitespace",
			input: "  main  \n  develop  \n",
			want:  []string{"main", "develop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLines(tt.input)

			if len(got) != len(tt.want) {
				t.Fatalf("parseLines(%q) returned %d lines, want %d\ngot: %v\nwant: %v",
					tt.input, len(got), len(tt.want), got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseLines(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
