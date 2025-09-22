package main

import (
	"os"
	"testing"
)

func TestFileExists(t *testing.T) {
	// 存在するファイルのテスト
	tempFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if !fileExists(tempFile.Name()) {
		t.Error("存在するファイルがfalseを返した")
	}

	// 存在しないファイルのテスト
	if fileExists("/non/existent/file") {
		t.Error("存在しないファイルがtrueを返した")
	}
}

func TestIsCommandAvailable(t *testing.T) {
	// 一般的に存在するコマンドのテスト
	if !isCommandAvailable("echo") {
		t.Error("echoコマンドが利用できない")
	}

	// 存在しないコマンドのテスト
	if isCommandAvailable("nonexistentcommand12345") {
		t.Error("存在しないコマンドがtrueを返した")
	}
}

func TestPackageStruct(t *testing.T) {
	pkg := Package{
		Name:    "test-package",
		Version: "1.0.0",
	}

	if pkg.Name != "test-package" {
		t.Errorf("期待値: test-package, 実際: %s", pkg.Name)
	}

	if pkg.Version != "1.0.0" {
		t.Errorf("期待値: 1.0.0, 実際: %s", pkg.Version)
	}
}

func TestCheckerStruct(t *testing.T) {
	checker := &Checker{
		PackageManager: "npm",
		Packages: []Package{
			{Name: "lodash", Version: "4.17.21"},
		},
	}

	if checker.PackageManager != "npm" {
		t.Errorf("期待値: npm, 実際: %s", checker.PackageManager)
	}

	if len(checker.Packages) != 1 {
		t.Errorf("期待値: 1, 実際: %d", len(checker.Packages))
	}

	if checker.Packages[0].Name != "lodash" {
		t.Errorf("期待値: lodash, 実際: %s", checker.Packages[0].Name)
	}
}

func TestDetectPackageManager(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "pc-test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 現在のディレクトリを保存
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("現在のディレクトリの取得に失敗: %v", err)
	}
	defer os.Chdir(originalDir)

	// テスト用ディレクトリに移動
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("ディレクトリの変更に失敗: %v", err)
	}

	// pnpm-lock.yamlが存在する場合のテスト
	file, err := os.Create("pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("pnpm-lock.yamlの作成に失敗: %v", err)
	}
	file.Close()

	result := detectPackageManager()
	if result != "pnpm" {
		t.Errorf("pnpm-lock.yamlが存在する場合、期待値: pnpm, 実際: %s", result)
	}

	// ファイルを削除
	os.Remove("pnpm-lock.yaml")

	// yarn.lockが存在する場合のテスト
	file, err = os.Create("yarn.lock")
	if err != nil {
		t.Fatalf("yarn.lockの作成に失敗: %v", err)
	}
	file.Close()

	result = detectPackageManager()
	if result != "yarn" {
		t.Errorf("yarn.lockが存在する場合、期待値: yarn, 実際: %s", result)
	}

	// ファイルを削除
	os.Remove("yarn.lock")

	// package-lock.jsonが存在する場合のテスト
	file, err = os.Create("package-lock.json")
	if err != nil {
		t.Fatalf("package-lock.jsonの作成に失敗: %v", err)
	}
	file.Close()

	result = detectPackageManager()
	if result != "npm" {
		t.Errorf("package-lock.jsonが存在する場合、期待値: npm, 実際: %s", result)
	}
}