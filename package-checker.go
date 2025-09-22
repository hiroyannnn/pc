package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/alecthomas/kong"
)

type Package struct {
	Name    string
	Version string
}

type Checker struct {
	PackageManager string
	Packages       []Package
}

// CLI引数の構造体定義
type CLI struct {
	File string `arg:"" optional:"" type:"path" help:"パッケージリストファイル (省略時はデフォルトリストを使用)"`
	PM   string `short:"p" long:"pm" default:"auto" enum:"npm,pnpm,yarn,auto" help:"パッケージマネージャー (npm/pnpm/yarn/auto)"`
}

// デフォルトのパッケージリスト
var defaultPackages = []Package{
	{Name: "backslash", Version: "0.2.1"},
	{Name: "chalk-template", Version: "1.1.1"},
	{Name: "supports-hyperlinks", Version: "4.1.1"},
	{Name: "has-ansi", Version: "6.0.1"},
	{Name: "simple-swizzle", Version: "0.2.3"},
	{Name: "color-string", Version: "2.1.1"},
	{Name: "error-ex", Version: "1.3.3"},
	{Name: "color-name", Version: "2.0.1"},
	{Name: "is-arrayish", Version: "0.3.3"},
	{Name: "slice-ansi", Version: "7.1.1"},
	{Name: "color-convert", Version: "3.1.1"},
	{Name: "wrap-ansi", Version: "9.0.1"},
	{Name: "ansi-regex", Version: "6.2.1"},
	{Name: "supports-color", Version: "10.2.1"},
	{Name: "strip-ansi", Version: "7.1.1"},
	{Name: "chalk", Version: "5.6.1"},
	{Name: "debug", Version: "4.4.2"},
	{Name: "ansi-styles", Version: "6.2.2"},
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("pc"),
		kong.Description("📦 Node.jsパッケージの依存関係チェックツール"),
		kong.UsageOnError(),
	)

	// パッケージマネージャーの自動検出
	packageManager := cli.PM
	if packageManager == "auto" {
		detected := detectPackageManager()
		if detected == "" {
			fmt.Println("❌ エラー: パッケージマネージャーを自動検出できませんでした")
			fmt.Println("npm または pnpm を明示的に指定してください")
			ctx.Exit(1)
		}
		packageManager = detected
		fmt.Printf("🔍 パッケージマネージャーを自動検出しました: %s\n", packageManager)
	}

	checker := &Checker{
		PackageManager: packageManager,
	}

	// ファイルが指定されていればファイルから読み込み、なければデフォルトを使用
	if cli.File != "" {
		if err := checker.loadPackages(cli.File); err != nil {
			fmt.Printf("❌ ファイル読み込みエラー: %v\n", err)
			ctx.Exit(1)
		}
		fmt.Printf("📂 ファイルからパッケージリストを読み込みました: %s\n", cli.File)
	} else {
		checker.Packages = defaultPackages
		fmt.Println("📦 デフォルトのパッケージリストを使用します")
	}

	checker.checkPackages()
}

// パッケージマネージャーの自動検出
func detectPackageManager() string {
	// 1. ロックファイルの存在確認（優先度順）
	if fileExists("pnpm-lock.yaml") {
		return "pnpm"
	}
	if fileExists("yarn.lock") {
		return "yarn"
	}
	if fileExists("package-lock.json") {
		return "npm"
	}

	// 2. node_modulesの特殊フォルダ確認
	if fileExists("node_modules/.pnpm") {
		return "pnpm"
	}
	if fileExists("node_modules/.yarn-integrity") {
		return "yarn"
	}

	// 3. package.jsonのpackageManagerフィールド確認
	if pm := getPackageManagerFromPackageJson(); pm != "" {
		return pm
	}

	// 4. インストール状況確認（優先度順）
	if isCommandAvailable("pnpm") {
		return "pnpm"
	} else if isCommandAvailable("yarn") {
		return "yarn"
	} else if isCommandAvailable("npm") {
		return "npm"
	}

	return ""
}

// ファイルの存在確認
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// コマンドが利用可能かチェック
func isCommandAvailable(command string) bool {
	cmd := exec.Command("which", command)
	err := cmd.Run()
	return err == nil
}

// package.json から packageManager フィールドを取得
func getPackageManagerFromPackageJson() string {
	if !fileExists("package.json") {
		return ""
	}
	
	file, err := os.Open("package.json")
	if err != nil {
		return ""
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "\"packageManager\"") {
			// "packageManager": "pnpm@8.6.0" のような形式から抽出
			if strings.Contains(line, "pnpm") {
				return "pnpm"
			} else if strings.Contains(line, "yarn") {
				return "yarn"
			} else if strings.Contains(line, "npm") {
				return "npm"
			}
		}
	}
	
	return ""
}

func (c *Checker) loadPackages(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 1 {
			if strings.Contains(parts[0], "@") {
				pkgParts := strings.Split(parts[0], "@")
				c.Packages = append(c.Packages, Package{
					Name:    pkgParts[0],
					Version: pkgParts[1],
				})
			} else {
				c.Packages = append(c.Packages, Package{
					Name:    parts[0],
					Version: "",
				})
			}
		} else if len(parts) >= 2 {
			c.Packages = append(c.Packages, Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if len(c.Packages) == 0 {
		return fmt.Errorf("パッケージリストが空です")
	}

	return nil
}

func (c *Checker) checkPackages() {
	fmt.Printf("🔍 使用するパッケージマネージャー: %s\n", c.PackageManager)
	fmt.Println("パッケージの依存関係をチェックしています...")
	fmt.Println("=========================================")

	for _, pkg := range c.Packages {
		c.checkPackage(pkg)
	}

	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("チェック完了")
}

func (c *Checker) checkPackage(pkg Package) {
	fmt.Println()
	if pkg.Version != "" {
		fmt.Printf("📦 %s (%s)\n", pkg.Name, pkg.Version)
	} else {
		fmt.Printf("📦 %s\n", pkg.Name)
	}
	fmt.Println("-----------------------------------------")

	var cmd *exec.Cmd
	var notFoundPattern string

	if c.PackageManager == "pnpm" {
		cmd = exec.Command("pnpm", "why", "-r", pkg.Name)
		notFoundPattern = "✕ Couldn't find any"
	} else if c.PackageManager == "yarn" {
		cmd = exec.Command("yarn", "why", pkg.Name)
		notFoundPattern = "error Package"
	} else {
		cmd = exec.Command("npm", "ls", pkg.Name)
		notFoundPattern = "npm ERR!"
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil && strings.Contains(outputStr, notFoundPattern) {
		fmt.Println("❓ MISSING")
		return
	}

	// パッケージが見つからない場合の追加チェック
	if strings.Contains(outputStr, "Legend:") && !strings.Contains(outputStr, pkg.Name+" ") {
		fmt.Println("❓ MISSING")
		return
	}

	// パッケージマネージャー別のバージョンチェック
	if pkg.Version != "" {
		switch c.PackageManager {
		case "pnpm":
			c.checkPnpmVersion(pkg, outputStr)
		case "yarn":
			c.checkYarnVersion(pkg, outputStr)
		default:
			c.checkNpmVersion(pkg, outputStr)
		}
	} else if strings.Contains(outputStr, pkg.Name) {
		fmt.Println("📦 FOUND")
	} else {
		fmt.Println("❓ MISSING")
	}

	// パッケージ名を含む関連行のみ表示
	lines := strings.Split(outputStr, "\n")
	relevantLines := []string{}
	
	for _, line := range lines {
		if line != "" && strings.Contains(line, pkg.Name) {
			relevantLines = append(relevantLines, line)
		}
	}
	
	if len(relevantLines) > 0 {
		fmt.Println("関連する依存関係:")
		for _, line := range relevantLines {
			fmt.Printf("  %s\n", line)
		}
	}
}

// pnpm用のバージョンチェック
func (c *Checker) checkPnpmVersion(pkg Package, outputStr string) {
	// パターン1: └── パッケージ名 バージョン
	// パターン2: │ └── パッケージ名 バージョン
	// パターン3: └─┬ パッケージ名 バージョン
	patterns := []string{
		fmt.Sprintf(`[└├]─+.*%s %s`, regexp.QuoteMeta(pkg.Name), regexp.QuoteMeta(pkg.Version)),
		fmt.Sprintf(`%s %s$`, regexp.QuoteMeta(pkg.Name), regexp.QuoteMeta(pkg.Version)),
	}

	var matched bool
	for _, pattern := range patterns {
		if matched, _ = regexp.MatchString(pattern, outputStr); matched {
			break
		}
	}

	if matched {
		fmt.Printf("🎯 MATCH: %s\n", pkg.Version)
	} else {
		// 実際のバージョンを取得 - より幅広いパターンに対応
		versionPatterns := []string{
			fmt.Sprintf(`[└├│]─+.*%s ([0-9]+\.[0-9]+\.[0-9]+[^\s]*)`, regexp.QuoteMeta(pkg.Name)),
			fmt.Sprintf(`%s ([0-9]+\.[0-9]+\.[0-9]+[^\s]*)`, regexp.QuoteMeta(pkg.Name)),
		}

		var versions []string
		for _, versionPattern := range versionPatterns {
			re := regexp.MustCompile(versionPattern)
			matches := re.FindAllStringSubmatch(outputStr, -1)

			for _, match := range matches {
				if len(match) > 1 {
					versions = append(versions, match[1])
				}
			}
		}

		if len(versions) > 0 {
			// 重複を除去
			uniqueVersions := make(map[string]bool)
			for _, v := range versions {
				uniqueVersions[v] = true
			}
			var uniqueList []string
			for v := range uniqueVersions {
				uniqueList = append(uniqueList, v)
			}
			fmt.Printf("📦 FOUND: %s\n", strings.Join(uniqueList, ", "))
		} else {
			fmt.Println("❓ MISSING")
		}
	}
}

// yarn用のバージョンチェック
func (c *Checker) checkYarnVersion(pkg Package, outputStr string) {
	// yarn whyの出力形式:
	// └─ パッケージ名@バージョン
	// ├─ パッケージ名@バージョン
	exactPattern := fmt.Sprintf(`[└├]─ %s@%s`, regexp.QuoteMeta(pkg.Name), regexp.QuoteMeta(pkg.Version))

	if matched, _ := regexp.MatchString(exactPattern, outputStr); matched {
		fmt.Printf("🎯 MATCH: %s\n", pkg.Version)
	} else {
		// 実際のバージョンを取得
		versionPattern := fmt.Sprintf(`[└├]─ %s@([0-9]+\.[0-9]+\.[0-9]+[^\s]*)`, regexp.QuoteMeta(pkg.Name))
		re := regexp.MustCompile(versionPattern)
		matches := re.FindAllStringSubmatch(outputStr, -1)

		var versions []string
		for _, match := range matches {
			if len(match) > 1 {
				versions = append(versions, match[1])
			}
		}

		if len(versions) > 0 {
			// 重複を除去
			uniqueVersions := make(map[string]bool)
			for _, v := range versions {
				uniqueVersions[v] = true
			}
			var uniqueList []string
			for v := range uniqueVersions {
				uniqueList = append(uniqueList, v)
			}
			fmt.Printf("📦 FOUND: %s\n", strings.Join(uniqueList, ", "))
		} else {
			fmt.Println("❓ MISSING")
		}
	}
}

// npm用のバージョンチェック
func (c *Checker) checkNpmVersion(pkg Package, outputStr string) {
	if strings.Contains(outputStr, pkg.Version) {
		fmt.Printf("🎯 MATCH: %s\n", pkg.Version)
	} else {
		fmt.Println("📦 FOUND: 異なるバージョン")
	}
}