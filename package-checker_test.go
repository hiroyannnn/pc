package main

import (
	"context"
	"os"
	"testing"
)

func closeTestFile(t *testing.T, file *os.File) {
	t.Helper()
	if err := file.Close(); err != nil {
		t.Fatalf("ファイルのクローズに失敗: %v", err)
	}
}

func removeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("ファイルの削除に失敗: %v", err)
	}
}

func cleanupDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("ディレクトリの削除に失敗: %v", err)
	}
}

func changeDirectory(t *testing.T, path string) {
	t.Helper()
	if err := os.Chdir(path); err != nil {
		t.Fatalf("ディレクトリの変更に失敗: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatalf("一時ファイルの作成に失敗: %v", err)
	}
	t.Cleanup(func() {
		removeTestFile(t, tempFile.Name())
	})
	t.Cleanup(func() {
		closeTestFile(t, tempFile)
	})

	if !fileExists(tempFile.Name()) {
		t.Error("存在するファイルがfalseを返した")
	}

	if fileExists("/non/existent/file") {
		t.Error("存在しないファイルがtrueを返した")
	}
}

func TestIsCommandAvailable(t *testing.T) {
	if !isCommandAvailable(os.Args[0]) {
		t.Error("実行中のテストファイルを検出できない")
	}

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
		PackageManager: packageManagerNPM,
		Packages: []Package{
			{Name: "lodash", Version: "4.17.21"},
		},
	}

	if checker.PackageManager != packageManagerNPM {
		t.Errorf("期待値: %s, 実際: %s", packageManagerNPM, checker.PackageManager)
	}

	if len(checker.Packages) != 1 {
		t.Errorf("期待値: 1, 実際: %d", len(checker.Packages))
	}

	if checker.Packages[0].Name != "lodash" {
		t.Errorf("期待値: lodash, 実際: %s", checker.Packages[0].Name)
	}
}

func TestHasMissingPackageOutput(t *testing.T) {
	tests := []struct {
		name            string
		err             error
		output          string
		notFoundPattern string
		packageName     string
		want            bool
	}{
		{
			name:            "command error matches missing message",
			err:             os.ErrNotExist,
			output:          "npm ERR! missing package",
			notFoundPattern: "npm ERR!",
			packageName:     "lodash",
			want:            true,
		},
		{
			name:            "legend omits package",
			output:          "Legend:\n  lodash 4.17.21",
			notFoundPattern: "missing",
			packageName:     "left-pad",
			want:            true,
		},
		{
			name:            "legend includes package",
			output:          "Legend:\n  lodash 4.17.21",
			notFoundPattern: "missing",
			packageName:     "lodash",
			want:            false,
		},
		{
			name:            "normal output",
			output:          "lodash 4.17.21",
			notFoundPattern: "missing",
			packageName:     "lodash",
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasMissingPackageOutput(tt.err, tt.output, tt.notFoundPattern, tt.packageName); got != tt.want {
				t.Errorf("hasMissingPackageOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPackagesCanceledBeforeStarting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := &Checker{PackageManager: packageManagerNPM}
	if checker.checkPackages(ctx) {
		t.Fatal("canceled package check reported completion")
	}
}

func TestDetectPackageManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pc-test")
	if err != nil {
		t.Fatalf("一時ディレクトリの作成に失敗: %v", err)
	}
	t.Cleanup(func() {
		cleanupDirectory(t, tempDir)
	})

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("現在のディレクトリの取得に失敗: %v", err)
	}
	t.Cleanup(func() {
		changeDirectory(t, originalDir)
	})

	changeDirectory(t, tempDir)

	file, err := os.Create("pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("pnpm-lock.yamlの作成に失敗: %v", err)
	}
	closeTestFile(t, file)

	result := detectPackageManager()
	if result != packageManagerPNPM {
		t.Errorf("pnpm-lock.yamlが存在する場合、期待値: %s, 実際: %s", packageManagerPNPM, result)
	}

	removeTestFile(t, "pnpm-lock.yaml")

	file, err = os.Create("yarn.lock")
	if err != nil {
		t.Fatalf("yarn.lockの作成に失敗: %v", err)
	}
	closeTestFile(t, file)

	result = detectPackageManager()
	if result != packageManagerYarn {
		t.Errorf("yarn.lockが存在する場合、期待値: %s, 実際: %s", packageManagerYarn, result)
	}

	removeTestFile(t, "yarn.lock")

	file, err = os.Create("package-lock.json")
	if err != nil {
		t.Fatalf("package-lock.jsonの作成に失敗: %v", err)
	}
	closeTestFile(t, file)

	result = detectPackageManager()
	if result != packageManagerNPM {
		t.Errorf("package-lock.jsonが存在する場合、期待値: %s, 実際: %s", packageManagerNPM, result)
	}
}
