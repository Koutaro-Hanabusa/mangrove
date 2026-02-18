package command

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Koutaro-Hanabusa/mangrove"
	"github.com/spf13/cobra"
)

var (
	applyYes    bool
	applyRepos  []string
	applyMethod string
	applyBase   string
	applyBranch string
)

var applyCmd = &cobra.Command{
	Use:   "apply [workspace-name]",
	Short: "ワークツリーの変更を元リポジトリに反映する",
	Long: `ワークスペースのワークツリーで行った変更を元のリポジトリに反映する。

反映方法:
  stash  - ワークツリーの未コミット変更をstashし、元リポの新ブランチでpopする
  merge  - ワークツリーブランチを元リポの新ブランチにマージする

使用例:
  mgv apply
  mgv apply feature-login
  mgv apply feature-login --method stash --base main --branch apply/feature-login
  mgv apply feature-login --repo api --repo web
  mgv apply feature-login -y -m merge -b main`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		interactive := !applyYes

		var profileName, wsName string

		if len(args) > 0 {
			wsName = args[0]
			_, pName, err := resolveProfile(profileFlag == "")
			if err != nil {
				return err
			}
			profileName = pName
		} else {
			if !mangrove.IsFzfAvailable() {
				return fmt.Errorf("fzf is required for interactive mode. Install with: brew install fzf")
			}

			workspaces, err := mangrove.ListWorkspaces(cfg, profileFlag)
			if err != nil {
				return err
			}
			if len(workspaces) == 0 {
				return fmt.Errorf("no workspaces found")
			}

			labels := mangrove.WorkspaceLabels(workspaces)
			selected, err := mangrove.SelectWorkspace(labels)
			if err != nil {
				return err
			}

			pName, wName, err := mangrove.ParseWorkspaceLabel(selected)
			if err != nil {
				return err
			}
			profileName = pName
			wsName = wName
		}

		profile, _, err := cfg.GetProfile(profileName)
		if err != nil {
			return err
		}

		wsPath := mangrove.GetWorkspacePath(cfg, profileName, wsName)

		// --repo で対象リポを絞り込む
		repoFilter := make(map[string]bool)
		for _, r := range applyRepos {
			repoFilter[r] = true
		}

		fmt.Fprintf(os.Stderr, "\nApplying workspace: %s/%s\n",
			mangrove.ProfileNameStyle.Render(profileName),
			mangrove.RepoNameStyle.Render(wsName),
		)

		for _, repo := range profile.Repos {
			if len(repoFilter) > 0 && !repoFilter[repo.Name] {
				continue
			}

			wtDir := filepath.Join(wsPath, repo.Name)
			if _, err := os.Stat(wtDir); os.IsNotExist(err) {
				mangrove.PrintWarning("%s: worktree not found, skipping", repo.Name)
				continue
			}

			fmt.Fprintf(os.Stderr, "\n[%s]\n", mangrove.RepoNameStyle.Render(repo.Name))

			// ステータス表示
			branch, err := mangrove.CurrentBranch(wtDir)
			if err != nil {
				mangrove.PrintError("%s: failed to get branch: %v", repo.Name, err)
				continue
			}

			changedCount, err := mangrove.StatusChangedCount(wtDir)
			if err != nil {
				mangrove.PrintError("%s: failed to get status: %v", repo.Name, err)
				continue
			}

			ahead, behind, _ := mangrove.AheadBehind(repo.Path, repo.GetDefaultBase(), branch)
			mangrove.PrintRepoStatus(repo.Name, branch, changedCount, ahead, behind, repo.GetDefaultBase())

			// ガード: 元リポに未コミット変更があればスキップ
			origStatus, err := mangrove.StatusPorcelain(repo.Path)
			if err != nil {
				mangrove.PrintError("%s: failed to check original repo status: %v", repo.Name, err)
				continue
			}
			if origStatus != "" {
				mangrove.PrintError("%s: 元リポに未コミット変更があります。先にcommitかstashしてください。", repo.Name)
				continue
			}

			// 反映方法の選択
			method := applyMethod
			if method == "" {
				if interactive {
					if !mangrove.IsFzfAvailable() {
						return fmt.Errorf("fzf is required for interactive mode")
					}
					selected, err := mangrove.SelectMethod(repo.Name)
					if err != nil {
						return err
					}
					method = selected
				} else {
					return fmt.Errorf("--method is required in non-interactive mode")
				}
			}

			if method == "skip" {
				mangrove.PrintInfo("Skipped %s", repo.Name)
				continue
			}

			// ガード: stashには未コミット変更が必要
			if method == "stash" && changedCount == 0 {
				mangrove.PrintWarning("%s: 未コミット変更がないためスキップします", repo.Name)
				continue
			}

			// ガード: mergeにはaheadなコミットが必要
			if method == "merge" && ahead == 0 {
				mangrove.PrintWarning("%s: マージ対象のコミットがないためスキップします", repo.Name)
				continue
			}

			// ベースブランチの選択
			baseBranch := applyBase
			if baseBranch == "" {
				if interactive {
					prompt := fmt.Sprintf("[%s] Base branch:", repo.Name)
					selected, err := mangrove.SelectBranch(repo.Path, prompt, repo.GetDefaultBase())
					if err != nil {
						return err
					}
					baseBranch = selected
				} else {
					baseBranch = repo.GetDefaultBase()
				}
			}

			// 新ブランチ名の決定
			newBranch := applyBranch
			if newBranch == "" {
				defaultName := fmt.Sprintf("apply/%s", wsName)
				if interactive {
					fmt.Fprintf(os.Stderr, "  ? New branch name [%s]: ", defaultName)
					reader := bufio.NewReader(os.Stdin)
					input, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("failed to read branch name: %w", err)
					}
					input = strings.TrimSpace(input)
					if input != "" {
						newBranch = input
					} else {
						newBranch = defaultName
					}
				} else {
					newBranch = defaultName
				}
			}

			// 実行
			switch method {
			case "stash":
				if err := applyStash(wtDir, repo.Path, newBranch, baseBranch, repo.Name); err != nil {
					mangrove.PrintError("%s: stash apply failed: %v", repo.Name, err)
					continue
				}
			case "merge":
				if err := applyMerge(wtDir, repo.Path, branch, newBranch, baseBranch, repo.Name); err != nil {
					mangrove.PrintError("%s: merge apply failed: %v", repo.Name, err)
					continue
				}
			default:
				mangrove.PrintError("%s: unknown method %q", repo.Name, method)
				continue
			}

			mangrove.PrintSuccess("%s: applied via %s → %s (base: %s)", repo.Name, method, newBranch, baseBranch)
		}

		fmt.Fprintln(os.Stderr)
		return nil
	},
}

// applyStash はstashでワークツリーの未コミット変更を元リポに反映する。
// stash reflogはworktreeごとに独立なので、SHA経由でオブジェクトを共有する。
func applyStash(wtDir, repoPath, newBranch, baseBranch, repoName string) error {
	// worktreeでstash push
	msg := fmt.Sprintf("mgv-apply: %s", newBranch)
	if err := mangrove.StashPush(wtDir, msg); err != nil {
		return fmt.Errorf("stash push failed: %w", err)
	}

	// stashのSHAを取得（worktreeのreflogから）
	stashRef, err := mangrove.StashRef(wtDir)
	if err != nil {
		// ロールバック: worktreeでstash popして復元
		mangrove.PrintWarning("%s: ロールバック中...", repoName)
		_ = mangrove.StashPop(wtDir)
		return fmt.Errorf("stash ref取得に失敗: %w", err)
	}

	// 元リポで新ブランチを作成
	if err := mangrove.CheckoutNewBranch(repoPath, newBranch, baseBranch); err != nil {
		// ロールバック: worktreeでstash popして復元
		mangrove.PrintWarning("%s: ロールバック中...", repoName)
		_ = mangrove.StashPop(wtDir)
		return fmt.Errorf("checkout -b failed: %w", err)
	}

	// 元リポでstash apply（SHAを指定してオブジェクトストア経由で適用）
	if err := mangrove.StashApply(repoPath, stashRef); err != nil {
		// ロールバック: ベースブランチに戻る → 新ブランチ削除 → worktreeでstash pop
		mangrove.PrintWarning("%s: ロールバック中...", repoName)
		_ = mangrove.CheckoutBranch(repoPath, baseBranch)
		_ = mangrove.BranchDelete(repoPath, newBranch, true)
		_ = mangrove.StashPop(wtDir)
		return fmt.Errorf("stash apply failed: %w", err)
	}

	// 成功したのでworktreeのstashを掃除
	_ = mangrove.StashDrop(wtDir)

	return nil
}

// applyMerge はワークツリーブランチを元リポの新ブランチにマージして反映する。
func applyMerge(wtDir, repoPath, wtBranch, newBranch, baseBranch, repoName string) error {
	// 元リポの現在ブランチを記録
	origBranch, err := mangrove.CurrentBranch(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// 元リポで新ブランチを作成
	if err := mangrove.CheckoutNewBranch(repoPath, newBranch, baseBranch); err != nil {
		return fmt.Errorf("checkout -b failed: %w", err)
	}

	// ワークツリーブランチをマージ
	if err := mangrove.Merge(repoPath, wtBranch); err != nil {
		// ロールバック: merge中断 → 元ブランチに戻る → 新ブランチ削除
		mangrove.PrintWarning("%s: ロールバック中...", repoName)
		_ = mangrove.MergeAbort(repoPath)
		_ = mangrove.CheckoutBranch(repoPath, origBranch)
		_ = mangrove.BranchDelete(repoPath, newBranch, true)
		return fmt.Errorf("merge failed: %w", err)
	}

	// 元のブランチに復帰
	if err := mangrove.CheckoutBranch(repoPath, origBranch); err != nil {
		mangrove.PrintWarning("%s: %s への復帰に失敗: %v", repoName, origBranch, err)
	}

	return nil
}

func init() {
	applyCmd.Flags().BoolVarP(&applyYes, "yes", "y", false, "非インタラクティブモード")
	applyCmd.Flags().StringSliceVarP(&applyRepos, "repo", "r", nil, "対象リポを絞る")
	applyCmd.Flags().StringVarP(&applyMethod, "method", "m", "", "反映方法: stash or merge")
	applyCmd.Flags().StringVarP(&applyBase, "base", "b", "", "ベースブランチ")
	applyCmd.Flags().StringVar(&applyBranch, "branch", "", "新ブランチ名")
	rootCmd.AddCommand(applyCmd)
}
