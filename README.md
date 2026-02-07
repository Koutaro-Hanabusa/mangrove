# mangrove (mgv)

**複数リポジトリの git worktree をまとめて管理する CLI ツール**

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

## 概要

`mgv` は、複数の Git リポジトリにまたがる開発ワークスペースを `git worktree` で一括管理するコマンドラインツールです。

フロントエンドとバックエンドなど、複数リポジトリをセットで並行開発する際に、以下の課題を解決します。

- 各リポで個別に `git worktree add/remove` するのが面倒
- リポごとに派生元ブランチが異なる（frontend は `main` から、backend は `develop` からなど）
- 「この作業はどのリポのどのブランチだったか」を把握しづらい

`mgv` では **プロファイル** にリポジトリの組み合わせを定義し、**ワークスペース** としてまとめて worktree を作成・削除・管理できます。fzf を使った対話式のブランチ選択により、操作もスムーズです。

## インストール

### go install

```bash
go install github.com/1126buri/mangrove/cmd/mgv@latest
```

### ソースからビルド

```bash
git clone https://github.com/1126buri/mangrove.git
cd mangrove
go build -o mgv ./cmd/mgv
```

ビルドされた `mgv` バイナリを `$PATH` の通った場所に配置してください。

## 前提条件

| ツール | 必須/任意 | 用途 |
|--------|----------|------|
| `git` | 必須 | worktree 操作全般 |
| `fzf` | 対話モードで必須、`--yes` 時は不要 | ブランチ/ワークスペースの選択 |

fzf のインストール:

```bash
# macOS
brew install fzf

# Linux
sudo apt install fzf
```

## 設定

設定ファイルの場所: `~/.config/mgv/config.yaml`

```yaml
base_dir: ~/mgv-workspaces
default_profile: project-a

profiles:
  project-a:
    repos:
      - name: frontend-A
        path: ~/repos/frontend-A
        default_base: main
      - name: backend
        path: ~/repos/backend
        default_base: develop
    hooks:
      post_create:
        - repo: frontend-A
          run: npm install
        - repo: backend
          run: go mod download

  project-b:
    repos:
      - name: frontend-B
        path: ~/repos/frontend-B
      - name: backend
        path: ~/repos/backend
        default_base: develop
```

### 設定項目

| キー | 説明 | デフォルト値 |
|------|------|-------------|
| `base_dir` | ワークスペースの親ディレクトリ | `~/mgv-workspaces` |
| `default_profile` | `--profile` 省略時に使われるプロファイル | (なし) |
| `profiles` | プロファイルの定義 | `{}` |
| `profiles.*.repos[].name` | リポジトリの表示名 (worktree ディレクトリ名にも使用) | |
| `profiles.*.repos[].path` | ベアリポジトリまたはクローン済みリポジトリのパス | |
| `profiles.*.repos[].default_base` | 派生元のデフォルトブランチ | `main` |
| `profiles.*.hooks.post_create` | ワークスペース作成後に実行するフック | `[]` |

## 使い方

すべてのコマンドで `--profile` (`-p`) フラグを使ってプロファイルを指定できます。省略時は `default_profile` が使用されます。

### `mgv new` - ワークスペースの作成

対話モードでは、プロファイル選択、ワークスペース名の入力、各リポの派生元ブランチを fzf で選択できます。

```bash
# 対話モード (fzf でブランチ選択)
mgv new

# ワークスペース名を指定して対話モード
mgv new feature-login

# 非対話モード (default_base を自動使用)
mgv new feature-login --yes

# プロファイルを指定して非対話モード
mgv new feature-login --profile project-a --yes

# 全リポ共通で派生元ブランチを指定
mgv new feature-login --base develop --yes
```

**フラグ:**

| フラグ | 短縮形 | 説明 |
|--------|-------|------|
| `--yes` | `-y` | 非対話モード (デフォルトブランチを自動使用) |
| `--base` | `-b` | 全リポ共通の派生元ブランチ |
| `--profile` | `-p` | 使用するプロファイル |

ワークスペース作成後、`hooks.post_create` に定義されたコマンドが各リポのディレクトリ内で実行されます。

### `mgv rm` - ワークスペースの削除

```bash
# 対話モード (fzf でワークスペース選択 → 確認)
mgv rm

# ワークスペース名を指定
mgv rm feature-login

# ブランチも一緒に削除
mgv rm feature-login --with-branch

# 非対話モード (確認なし)
mgv rm feature-login --profile project-a --with-branch --yes

# 未コミット変更があっても強制削除
mgv rm feature-login --force --yes
```

**フラグ:**

| フラグ | 短縮形 | 説明 |
|--------|-------|------|
| `--yes` | `-y` | 非対話モード (確認をスキップ) |
| `--with-branch` | | ローカルブランチも合わせて削除 |
| `--force` | `-f` | 未コミット変更があっても強制削除 |
| `--profile` | `-p` | 使用するプロファイル |

対話モードでは、未コミット変更がある場合に警告を表示し、強制削除するかどうか確認します。

### `mgv list` - ワークスペースの一覧

エイリアス: `mgv ls`

```bash
# 全プロファイルのワークスペースを一覧表示
mgv list

# プロファイルで絞り込み
mgv list --profile project-a
```

出力例:

```
project-a:
  feature-login      [frontend-A: ✓ clean] [backend: ● 2 changed]
  feature-payment    [frontend-A: ✓ clean] [backend: ✓ clean]

project-b:
  hotfix-123         [frontend-B: ✓ clean] [backend: ✓ clean]
```

### `mgv cd` - ワークスペースへの移動

ワークスペースのパスを標準出力に出力します。`cd` と組み合わせて使用します。

```bash
# 対話モード (fzf でワークスペース選択)
cd $(mgv cd)

# ワークスペース名を直接指定
cd $(mgv cd feature-login)

# プロファイルも指定
cd $(mgv cd feature-login --profile project-a)
```

### `mgv exec` - ワークスペース内でのコマンド実行

ワークスペースの各リポジトリで同じコマンドを実行します。`--` の後にコマンドを記述します。

```bash
# 対話モード (fzf でワークスペース選択)
mgv exec -- git status

# ワークスペース名を指定
mgv exec feature-login -- git status

# プロファイルも指定
mgv exec feature-login --profile project-a -- make build
```

出力例:

```
[frontend-A]
On branch feature-login
nothing to commit, working tree clean

[backend]
On branch feature-login
Changes not staged for commit:
  modified: main.go
```

### `mgv status` - 詳細なステータス表示

各リポのブランチ名、変更状態、ahead/behind を表示します。

```bash
# 対話モード (fzf でワークスペース選択)
mgv status

# ワークスペース名を指定
mgv status feature-login

# プロファイルも指定
mgv status feature-login --profile project-a
```

出力例:

```
project-a/feature-login:
  frontend-A        feature-login  ✓ clean      (2 commits ahead of main)
  backend           feature-login  ● 3 changed  (1 commit ahead of develop)
```

### `mgv profile` - プロファイルの管理

```bash
# プロファイル一覧 (* はデフォルト)
mgv profile list

# プロファイルの詳細表示
mgv profile show project-a
```

`profile list` の出力例:

```
 * project-a  (2 repos)
   project-b  (2 repos)
```

`profile show` の出力例:

```
project-a (default)

  Repositories
    frontend-A
      path:         /Users/you/repos/frontend-A
      default_base: main
    backend
      path:         /Users/you/repos/backend
      default_base: develop

  Hooks (post_create)
    frontend-A: npm install
    backend: go mod download
```

## コマンドまとめ

| コマンド | 対話式 | 非対話 | 説明 |
|---------|--------|--------|------|
| `mgv new [name]` | profile / name / base branch を対話選択 | `--yes` `--base` `--profile` | ワークスペース作成 |
| `mgv rm [name]` | workspace 選択 / 確認 | `--yes` `--force` `--with-branch` `--profile` | ワークスペース削除 |
| `mgv list` | - | `--profile` | 一覧表示 |
| `mgv cd [name]` | fzf でワークスペース選択 | 引数で直接指定 | パス出力 |
| `mgv exec [name] -- cmd` | fzf でワークスペース選択 | 引数で直接指定 | 一括コマンド実行 |
| `mgv status [name]` | fzf でワークスペース選択 | 引数で直接指定 | git status まとめ表示 |
| `mgv profile list` | - | - | プロファイル一覧 |
| `mgv profile show <name>` | - | - | プロファイル詳細 |

## ディレクトリ構成

### ワークスペースのレイアウト

```
{base_dir}/
├── {profile}/
│   ├── {workspace-name}/
│   │   ├── {repo-name}/     ← git worktree
│   │   └── {repo-name}/     ← git worktree
│   └── {workspace-name}/
│       └── ...
└── {profile}/
    └── ...
```

例:

```
~/mgv-workspaces/
├── project-a/
│   ├── feature-login/
│   │   ├── frontend-A/      ← git worktree (main → feature-login)
│   │   └── backend/         ← git worktree (develop → feature-login)
│   └── feature-payment/
│       ├── frontend-A/
│       └── backend/
└── project-b/
    └── hotfix-123/
        ├── frontend-B/
        └── backend/
```

### プロジェクトのソース構成

```
mangrove/
├── cmd/
│   └── mgv/
│       └── main.go          # エントリーポイント
├── command/
│   ├── root.go              # ルートコマンド + グローバルフラグ
│   ├── new.go               # mgv new
│   ├── rm.go                # mgv rm
│   ├── list.go              # mgv list
│   ├── cd.go                # mgv cd
│   ├── exec.go              # mgv exec
│   ├── status.go            # mgv status
│   └── profile.go           # mgv profile list / show
├── config.go                # 設定読み込み、Profile / Repo 構造体
├── git.go                   # git コマンド呼び出しラッパー
├── workspace.go             # ワークスペース操作ロジック
├── fzf.go                   # fzf 呼び出しヘルパー
├── ui.go                    # lipgloss スタイル定義、出力ヘルパー
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

## ライセンス

[MIT License](./LICENSE) - Copyright (c) 2026 1126buri
