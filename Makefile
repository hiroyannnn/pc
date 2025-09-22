.PHONY: help build test lint clean install cross-build

# デフォルトターゲット
help: ## ヘルプを表示
	@echo "利用可能なコマンド:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## バイナリをビルド
	go build -o pc package-checker.go

test: ## テストを実行
	go test -v -race -coverprofile=coverage.out ./...

lint: ## Lintチェックを実行
	golangci-lint run

clean: ## ビルド成果物を削除
	rm -f pc pc-* coverage.out
	rm -rf dist/

install: ## pcコマンドをインストール
	go install

cross-build: ## クロスプラットフォームビルド
	mkdir -p dist
	# Linux
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/pc-linux-amd64 package-checker.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/pc-linux-arm64 package-checker.go
	# macOS
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/pc-darwin-amd64 package-checker.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/pc-darwin-arm64 package-checker.go
	# Windows
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/pc-windows-amd64.exe package-checker.go
	GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o dist/pc-windows-arm64.exe package-checker.go

dev: build ## 開発用ビルド
	./pc --help

coverage: test ## テストカバレッジを表示
	go tool cover -html=coverage.out -o coverage.html
	@echo "カバレッジレポートを coverage.html に生成しました"

deps: ## 依存関係を更新
	go mod tidy
	go mod download

check: test lint ## テストとLintを実行

all: clean deps test lint build ## 全てのチェックとビルドを実行