package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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

type CLI struct {
	File string `arg:"" optional:"" type:"path" help:"パッケージリストファイル (省略時はデフォルトリストを使用)"`
	PM   string `short:"p" long:"pm" default:"auto" enum:"npm,pnpm,yarn,auto" help:"パッケージマネージャー (npm/pnpm/yarn/auto)"`
}

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

const (
	packageManagerPNPM = "pnpm"
	packageManagerYarn = "yarn"
	packageManagerNPM  = "npm"

	minPackageParts     = 2
	interruptedExitCode = 130 // 128 + SIGINT, as shells conventionally report interrupted commands.
)

func closeResource(closer io.Closer, name string) {
	if err := closer.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ リソースのクローズに失敗しました (%s): %v\n", name, err)
	}
}

func main() {
	var cli CLI
	cliCtx := kong.Parse(&cli,
		kong.Name("pc"),
		kong.Description("📦 Node.jsパッケージの依存関係チェックツール"),
		kong.UsageOnError(),
	)

	packageManager := cli.PM
	if packageManager == "auto" {
		detected := detectPackageManager()
		if detected == "" {
			fmt.Println("❌ エラー: パッケージマネージャーを自動検出できませんでした")
			fmt.Println("npm または pnpm を明示的に指定してください")
			cliCtx.Exit(1)
		}
		packageManager = detected
		fmt.Printf("🔍 パッケージマネージャーを自動検出しました: %s\n", packageManager)
	}

	checker := &Checker{
		PackageManager: packageManager,
	}

	if cli.File != "" {
		if err := checker.loadPackages(cli.File); err != nil {
			fmt.Printf("❌ ファイル読み込みエラー: %v\n", err)
			cliCtx.Exit(1)
		}
		fmt.Printf("📂 ファイルからパッケージリストを読み込みました: %s\n", cli.File)
	} else {
		checker.Packages = defaultPackages
		fmt.Println("📦 デフォルトのパッケージリストを使用します")
	}

	runContext, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if !checker.checkPackages(runContext) {
		cliCtx.Exit(interruptedExitCode)
	}
}

func detectPackageManager() string {
	if fileExists("pnpm-lock.yaml") {
		return packageManagerPNPM
	}
	if fileExists("yarn.lock") {
		return packageManagerYarn
	}
	if fileExists("package-lock.json") {
		return packageManagerNPM
	}

	if fileExists("node_modules/.pnpm") {
		return packageManagerPNPM
	}
	if fileExists("node_modules/.yarn-integrity") {
		return packageManagerYarn
	}

	if pm := getPackageManagerFromPackageJson(); pm != "" {
		return pm
	}

	switch {
	case isCommandAvailable(packageManagerPNPM):
		return packageManagerPNPM
	case isCommandAvailable(packageManagerYarn):
		return packageManagerYarn
	case isCommandAvailable(packageManagerNPM):
		return packageManagerNPM
	}

	return ""
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func getPackageManagerFromPackageJson() string {
	if !fileExists("package.json") {
		return ""
	}

	file, err := os.Open("package.json")
	if err != nil {
		return ""
	}
	defer closeResource(file, "package.json")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "\"packageManager\"") {
			switch {
			case strings.Contains(line, packageManagerPNPM):
				return packageManagerPNPM
			case strings.Contains(line, packageManagerYarn):
				return packageManagerYarn
			case strings.Contains(line, packageManagerNPM):
				return packageManagerNPM
			}
		}
	}

	return ""
}

func (c *Checker) loadPackages(filename string) error {
	cleanedFilename := filepath.Clean(filename)
	// #nosec G304 -- CLI の指定ファイルを読む機能のため、利用者が選んだパスを受け入れる
	file, err := os.Open(cleanedFilename)
	if err != nil {
		return err
	}
	defer closeResource(file, cleanedFilename)

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
		} else if len(parts) >= minPackageParts {
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

func hasMissingPackageOutput(err error, output, notFoundPattern, packageName string) bool {
	return err != nil && strings.Contains(output, notFoundPattern) ||
		strings.Contains(output, "Legend:") && !strings.Contains(output, packageName+" ")
}

func (c *Checker) checkPackages(ctx context.Context) bool {
	fmt.Printf("🔍 使用するパッケージマネージャー: %s\n", c.PackageManager)
	fmt.Println("パッケージの依存関係をチェックしています...")
	fmt.Println("=========================================")

	for _, pkg := range c.Packages {
		if ctx.Err() != nil {
			break
		}
		c.checkPackage(ctx, pkg)
	}

	if ctx.Err() != nil {
		fmt.Println("⏹️ チェックを中断しました")
		return false
	}

	fmt.Println()
	fmt.Println("=========================================")
	fmt.Println("チェック完了")
	return true
}

func (c *Checker) checkPackage(ctx context.Context, pkg Package) {
	fmt.Println()
	if pkg.Version != "" {
		fmt.Printf("📦 %s (%s)\n", pkg.Name, pkg.Version)
	} else {
		fmt.Printf("📦 %s\n", pkg.Name)
	}
	fmt.Println("-----------------------------------------")

	var cmd *exec.Cmd
	var notFoundPattern string

	switch c.PackageManager {
	case packageManagerPNPM:
		// #nosec G204 -- パッケージ名はシェル解釈を経ず、1つのコマンドライン引数として渡す
		cmd = exec.CommandContext(ctx, packageManagerPNPM, "why", "-r", pkg.Name)
		notFoundPattern = "✕ Couldn't find any"
	case packageManagerYarn:
		// #nosec G204 -- パッケージ名はシェル解釈を経ず、1つのコマンドライン引数として渡す
		cmd = exec.CommandContext(ctx, packageManagerYarn, "why", pkg.Name)
		notFoundPattern = "error Package"
	default:
		// #nosec G204 -- パッケージ名はシェル解釈を経ず、1つのコマンドライン引数として渡す
		cmd = exec.CommandContext(ctx, packageManagerNPM, "ls", pkg.Name)
		notFoundPattern = "npm ERR!"
	}

	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return
	}
	outputStr := string(output)

	if hasMissingPackageOutput(err, outputStr, notFoundPattern, pkg.Name) {
		fmt.Println("❓ MISSING")
		return
	}

	switch {
	case pkg.Version != "":
		switch c.PackageManager {
		case packageManagerPNPM:
			c.checkPnpmVersion(pkg, outputStr)
		case packageManagerYarn:
			c.checkYarnVersion(pkg, outputStr)
		default:
			c.checkNpmVersion(pkg, outputStr)
		}
	case strings.Contains(outputStr, pkg.Name):
		fmt.Println("📦 FOUND")
	default:
		fmt.Println("❓ MISSING")
	}

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

func (c *Checker) checkPnpmVersion(pkg Package, outputStr string) {
	patterns := []string{
		fmt.Sprintf(`[└├]─+.*%s %s`, regexp.QuoteMeta(pkg.Name), regexp.QuoteMeta(pkg.Version)),
		fmt.Sprintf(`%s %s$`, regexp.QuoteMeta(pkg.Name), regexp.QuoteMeta(pkg.Version)),
	}

	var matched bool
	for _, pattern := range patterns {
		var err error
		if matched, err = regexp.MatchString(pattern, outputStr); err != nil {
			continue
		}
		if matched {
			break
		}
	}

	if matched {
		fmt.Printf("🎯 MATCH: %s\n", pkg.Version)
	} else {
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

func (c *Checker) checkYarnVersion(pkg Package, outputStr string) {
	exactPattern := fmt.Sprintf(`[└├]─ %s@%s`, regexp.QuoteMeta(pkg.Name), regexp.QuoteMeta(pkg.Version))

	if matched, err := regexp.MatchString(exactPattern, outputStr); err == nil && matched {
		fmt.Printf("🎯 MATCH: %s\n", pkg.Version)
	} else {
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

func (c *Checker) checkNpmVersion(pkg Package, outputStr string) {
	if strings.Contains(outputStr, pkg.Version) {
		fmt.Printf("🎯 MATCH: %s\n", pkg.Version)
	} else {
		fmt.Println("📦 FOUND: 異なるバージョン")
	}
}
