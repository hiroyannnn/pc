# pc - Node.jsパッケージ依存関係チェックツール

[![CI](https://github.com/hiroyannnn/pc/actions/workflows/ci.yml/badge.svg)](https://github.com/hiroyannnn/pc/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hiroyannnn/pc)](https://goreportcard.com/report/github.com/hiroyannnn/pc)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

📦 Node.jsプロジェクトのパッケージ依存関係を効率的にチェックするコマンドラインツールです。

## 特徴

- 🔍 **自動パッケージマネージャー検出**: npm/pnpm/yarn を自動的に検出
- 📋 **バージョンマッチング**: 指定されたバージョンとの一致をチェック
- 📂 **ファイル入力対応**: パッケージリストファイルからの読み込み
- 🎯 **詳細な結果表示**: 依存関係ツリーの関連部分を表示

## インストール

### 必要な環境

- Go 1.23以降 (推奨: Go 1.25)

### バイナリリリースからインストール（推奨）

[GitHubリリースページ](https://github.com/hiroyannnn/pc/releases)から最新版をダウンロード：

```bash
# Linux (x86_64)
curl -L -o pc-linux-amd64.tar.gz https://github.com/hiroyannnn/pc/releases/latest/download/pc-linux-amd64.tar.gz
tar -xzf pc-linux-amd64.tar.gz
sudo mv pc-linux-amd64 /usr/local/bin/pc

# macOS (Apple Silicon)
curl -L -o pc-darwin-arm64.tar.gz https://github.com/hiroyannnn/pc/releases/latest/download/pc-darwin-arm64.tar.gz
tar -xzf pc-darwin-arm64.tar.gz
sudo mv pc-darwin-arm64 /usr/local/bin/pc

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/hiroyannnn/pc/releases/latest/download/pc-windows-amd64.zip" -OutFile "pc.zip"
Expand-Archive -Path "pc.zip" -DestinationPath "."
```

### ソースからインストール

```bash
# リポジトリをクローン
git clone https://github.com/hiroyannnn/pc.git
cd pc

# ビルド・インストール
go install

# PATHにGOPATH/binを追加（初回のみ）
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.zshrc
source ~/.zshrc
```

## 使用方法

### 基本的な使い方

```bash
# デフォルトパッケージリストでチェック
pc

# パッケージリストファイルを指定
pc packages.txt

# パッケージマネージャーを明示的に指定
pc -p npm
pc -p pnpm
pc -p yarn

# ヘルプ表示
pc --help
```

### パッケージリストファイル形式

以下の形式でパッケージリストファイルを作成できます：

```
# コメント行
chalk 5.6.1
debug@4.4.2
ansi-regex
lodash 4.17.21
```

## 出力例

```
🔍 パッケージマネージャーを自動検出しました: pnpm
📦 デフォルトのパッケージリストを使用します
🔍 使用するパッケージマネージャー: pnpm
パッケージの依存関係をチェックしています...
=========================================

📦 chalk (5.6.1)
-----------------------------------------
🎯 MATCH: 5.6.1
関連する依存関係:
  └── chalk 5.6.1

📦 debug (4.4.2)
-----------------------------------------
📦 FOUND: 4.3.4
関連する依存関係:
  ├── debug 4.3.4
```

## 結果の見方

- 🎯 **MATCH**: 指定されたバージョンと完全一致
- 📦 **FOUND**: パッケージは見つかったが異なるバージョン
- ❓ **MISSING**: パッケージが見つからない

## パッケージマネージャー自動検出

以下の順序で自動検出を行います：

1. ロックファイルの確認（優先度順）
   - `pnpm-lock.yaml` → pnpm
   - `yarn.lock` → yarn
   - `package-lock.json` → npm
2. node_modulesの特殊フォルダ確認
   - `node_modules/.pnpm` → pnpm
   - `node_modules/.yarn-integrity` → yarn
3. package.jsonの`packageManager`フィールド確認
4. インストール状況確認（優先度: pnpm > yarn > npm）

## オプション

```
Usage: pc [<file>] [flags]

Arguments:
  [<file>]    パッケージリストファイル (省略時はデフォルトリストを使用)

Flags:
  -h, --help         Show context-sensitive help.
  -p, --pm="auto"    パッケージマネージャー (npm/pnpm/yarn/auto)
```

## 開発

### Makefileを使用（推奨）

```bash
# ヘルプ表示
make help

# ビルド
make build

# テスト実行
make test

# Lintチェック
make lint

# 全チェック（テスト + Lint + ビルド）
make all

# クロスプラットフォームビルド
make cross-build
```

### 手動実行

```bash
# 開発用ビルド
go build -o pc package-checker.go

# テスト実行
go test ./...

# 依存関係の更新
go mod tidy
```

## ライセンス

MIT License

## 貢献

プルリクエストやイシューの報告を歓迎します！詳細は[CONTRIBUTING.md](CONTRIBUTING.md)をご覧ください。