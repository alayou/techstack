package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/urfave/cli/v3"

	"github.com/alayou/techstack/cmd/techstacktools/scanner"
	"github.com/alayou/techstack/cmd/techstacktools/sync"
	"github.com/alayou/techstack/cmd/techstacktools/tools"
)

// ScanFlags 扫描命令的选项
type ScanFlags struct {
	// 扫描选项
	Type         string // 包类型: all/golang/npm/pypi/cargo 或逗号分隔列表
	OutputFormat string // 输出格式: json/text/table
	ExportFile   string // 导出文件路径
	Verbose      bool   // 详细输出
	Dedupe       bool   // 去重（按 PURL）

	// 全局扫描选项
	Global   bool // 是否全局扫描
	ParseDep bool // 是否解析依赖文件 (go.mod/requirements.txt/Cargo.toml/package.json)

	// Cargo 特定选项
	Recursive     bool // 递归扫描子目录
	IncludeDev    bool // 包含开发依赖
	IncludeBuild  bool // 包含构建依赖
	GraphAnalysis bool // 构建依赖图
}

var scanFlags ScanFlags

func main() {
	app := &cli.Command{
		Name:  "techstacktools",
		Usage: "TechStack Tool CLI - scan and sync local packages to TechStack service",
		Commands: []*cli.Command{
			scanCommand(),
			syncCommand(),
			importCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// scanCommand 返回扫描命令
func scanCommand() *cli.Command {
	return &cli.Command{
		Name:      "scan",
		Aliases:   []string{"s"},
		Usage:     "Scan local packages",
		UsageText: "techstacktools scan [options] [dir]",
		Description: `Scan local packages from various package managers.
Supported types: golang, npm, pypi, cargo (comma-separated or 'all')

Global scanning will automatically detect:
  - npm: $(npm root -g)
  - pypi: $(python -m site --user-site) or /usr/local/lib/python*/site-packages
  - cargo: ~/.cargo/registry
  - golang: $GOPATH/pkg/mod

Use --parseDep to read dependency files (go.mod, requirements.txt, Cargo.toml, package.json)`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "type",
				Aliases:     []string{"t"},
				Usage:       "Package type: all/golang/npm/pypi/cargo (comma-separated supported)",
				DefaultText: "all",
				Value:       "all",
				Destination: &scanFlags.Type,
			},
			&cli.StringFlag{
				Name:        "format",
				Aliases:     []string{"f"},
				Usage:       "Output format: json/text/table",
				DefaultText: "json",
				Value:       "json",
				Destination: &scanFlags.OutputFormat,
			},
			&cli.StringFlag{
				Name:        "export",
				Aliases:     []string{"e", "o"},
				Usage:       "Export to file path",
				Destination: &scanFlags.ExportFile,
			},
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Usage:       "Verbose output",
				Destination: &scanFlags.Verbose,
			},
			&cli.BoolFlag{
				Name:        "dedupe",
				Aliases:     []string{"d"},
				Usage:       "Deduplicate by PURL (pkgtype:name)",
				Destination: &scanFlags.Dedupe,
			},
			&cli.BoolFlag{
				Name:        "global",
				Aliases:     []string{"g"},
				Usage:       "Scan global packages, ignore specified folder",
				Destination: &scanFlags.Global,
			},
			&cli.BoolFlag{
				Name:        "parseDep",
				Aliases:     []string{"p"},
				Usage:       "Parse dependency files (go.mod, requirements.txt, Cargo.toml, package.json)",
				Destination: &scanFlags.ParseDep,
			},
			// Cargo 特定选项
			&cli.BoolFlag{
				Name:        "recursive",
				Usage:       "Scan subdirectories recursively",
				Value:       true,
				Destination: &scanFlags.Recursive,
			},
			&cli.BoolFlag{
				Name:        "dev",
				Usage:       "Include dev dependencies",
				Destination: &scanFlags.IncludeDev,
			},
			&cli.BoolFlag{
				Name:        "build",
				Usage:       "Include build dependencies",
				Value:       true,
				Destination: &scanFlags.IncludeBuild,
			},
			&cli.BoolFlag{
				Name:        "graph",
				Usage:       "Build dependency graph",
				Destination: &scanFlags.GraphAnalysis,
			},
		},
		Action: runScan,
	}
}

// runScan 执行扫描
func runScan(ctx context.Context, cmd *cli.Command) error {
	// 获取位置参数 [dir]
	dir := ""
	if cmd.Args().Len() > 0 {
		dir = cmd.Args().First()
	}

	// 处理逗号分隔的类型
	types := parseTypes(scanFlags.Type)
	if scanFlags.Verbose {
		fmt.Printf("Scanning types: %v\n", types)
	}

	// 执行扫描
	var allPackages []scanner.Package

	// 首先处理全局扫描
	if scanFlags.Global {
		if scanFlags.Verbose {
			fmt.Println("Scanning global packages...")
		}
		globalPackages, err := scanGlobalPackages(types, scanFlags.ParseDep)
		if err != nil {
			if scanFlags.Verbose {
				fmt.Printf("Error scanning global: %v\n", err)
			}
		} else {
			allPackages = append(allPackages, globalPackages...)
			if scanFlags.Verbose {
				fmt.Printf("Found %d global packages\n", len(globalPackages))
			}
		}
	}

	// 如果指定了目录，扫描本地目录
	if dir != "" && !scanFlags.Global {
		path := filepath.Clean(dir)
		if scanFlags.Verbose {
			fmt.Printf("Scanning local directory: %s\n", path)
		}

		// 检查是否是依赖文件解析
		if scanFlags.ParseDep {
			localPackages, err := scanDependencyFiles(path, types)
			if err != nil {
				if scanFlags.Verbose {
					fmt.Printf("Error parsing dependency files: %v\n", err)
				}
			} else {
				allPackages = append(allPackages, localPackages...)
				if scanFlags.Verbose {
					fmt.Printf("Found %d packages from dependency files\n", len(localPackages))
				}
			}
		} else {
			// 标准扫描
			scanners, err := createScanners(types, path)
			if err != nil {
				return fmt.Errorf("creating scanners: %w", err)
			}

			for _, s := range scanners {
				if scanFlags.Verbose {
					fmt.Printf("Scanning %s packages in %s...\n", s.Name(), path)
				}

				pkgs, err := s.Scan()
				if err != nil {
					fmt.Printf("Error scanning %s: %v\n", s.Name(), err)
					continue
				}

				if scanFlags.Verbose {
					fmt.Printf("Found %d %s packages\n", len(pkgs), s.Name())
				}

				allPackages = append(allPackages, pkgs...)
			}
		}
	} else if dir == "" && !scanFlags.Global {
		// 没有指定目录且不是全局扫描，扫描当前目录
		path := "."
		if scanFlags.Verbose {
			fmt.Printf("Scanning local directory: %s\n", path)
		}

		scanners, err := createScanners(types, path)
		if err != nil {
			return fmt.Errorf("creating scanners: %w", err)
		}

		for _, s := range scanners {
			if scanFlags.Verbose {
				fmt.Printf("Scanning %s packages in %s...\n", s.Name(), path)
			}

			pkgs, err := s.Scan()
			if err != nil {
				fmt.Printf("Error scanning %s: %v\n", s.Name(), err)
				continue
			}

			if scanFlags.Verbose {
				fmt.Printf("Found %d %s packages\n", len(pkgs), s.Name())
			}

			allPackages = append(allPackages, pkgs...)
		}
	}

	// 去重（按 PURL 格式：pkgtype:name）
	if scanFlags.Dedupe {
		allPackages = deduplicateByPURL(allPackages)
		if scanFlags.Verbose {
			fmt.Printf("After deduplication: %d packages\n", len(allPackages))
		}
	}

	// 输出或导出
	if scanFlags.ExportFile != "" {
		exportPackages(allPackages, scanFlags.ExportFile, scanFlags.OutputFormat)
	} else {
		printPackages(allPackages, scanFlags.OutputFormat)
	}

	return nil
}

// scanGlobalPackages 扫描全局安装的包
func scanGlobalPackages(types []string, parseDep bool) ([]scanner.Package, error) {
	var packages []scanner.Package

	for _, t := range types {
		var pkgs []scanner.Package
		var err error

		switch t {
		case "npm":
			pkgs, err = scanNpmGlobal()
		case "golang":
			pkgs, err = scanGolangGlobal()
			// 如果启用了 parseDep，额外扫描 GOPATH/src 下的 go.mod 文件
			if parseDep && err == nil {
				gopath := os.Getenv("GOPATH")
				if gopath == "" {
					home := os.Getenv("HOME")
					if home != "" {
						gopath = filepath.Join(home, "go")
					}
				}
				if gopath != "" {
					srcPath := filepath.Join(gopath, "src")
					if _, err := os.Stat(srcPath); err == nil {
						depPkgs, _ := scanDependencyFiles(srcPath, []string{"golang"})
						pkgs = append(pkgs, depPkgs...)
					}
				}
			}
		case "pypi":
			pkgs, err = scanPypiGlobal()
		case "cargo":
			home := os.Getenv("HOME")
			pkgs, err = scanCargoGlobal(home)
		}

		if err != nil {
			continue
		}

		packages = append(packages, pkgs...)
	}

	return packages, nil
}

// scanNpmGlobal 扫描 npm 全局包
func scanNpmGlobal() ([]scanner.Package, error) {
	scanner := scanner.NewNpmScanner()
	return scanner.Scan()
}

// scanGolangGlobal 扫描 Go 全局包
func scanGolangGlobal() ([]scanner.Package, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home := os.Getenv("HOME")
		if home != "" {
			gopath = filepath.Join(home, "go")
		}
	}

	if gopath == "" {
		return nil, nil
	}

	scanner := scanner.NewGolangScanner()
	scanner.SetGopath(gopath)
	return scanner.Scan()
}

// scanPypiGlobal 扫描 PyPI 全局包
func scanPypiGlobal() ([]scanner.Package, error) {
	scanner := scanner.NewPypiScanner()
	return scanner.Scan()
}

// scanCargoGlobal 扫描 Cargo 全局包
func scanCargoGlobal(home string) ([]scanner.Package, error) {
	if home == "" {
		return nil, nil
	}

	cargoHome := filepath.Join(home, ".cargo", "registry")
	if _, err := os.Stat(cargoHome); err != nil {
		return nil, nil
	}

	scanner := scanner.NewCargoScanner(cargoHome)
	scanner.SetRecursive(true)
	return scanner.Scan()
}

// scanDependencyFiles 递归扫描目录下的依赖文件
func scanDependencyFiles(path string, types []string) ([]scanner.Package, error) {
	var packages []scanner.Package

	// 使用 filepath.Walk 递归遍历所有子目录
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录，只处理文件
		if info.IsDir() {
			return nil
		}

		name := info.Name()
		var pkgs []scanner.Package

		// 根据文件名匹配依赖文件
		switch name {
		case "go.mod":
			if containsType(types, "golang") {
				pkgs, _ = parseGoMod(filePath)
			}
		case "requirements.txt":
			if containsType(types, "pypi") {
				pkgs, _ = parseRequirementsTxt(filePath)
			}
		case "Cargo.toml":
			if containsType(types, "cargo") {
				pkgs, _ = parseCargoToml(filePath)
			}
		case "package.json":
			if containsType(types, "npm") {
				pkgs, _ = parsePackageJson(filePath)
			}
		}

		packages = append(packages, pkgs...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return packages, nil
}

// containsType 检查类型列表是否包含指定类型
func containsType(types []string, target string) bool {
	for _, t := range types {
		if t == target || t == "all" {
			return true
		}
	}
	return false
}

// parseGoMod 解析 go.mod 文件
func parseGoMod(path string) ([]scanner.Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packages []scanner.Package
	lines := strings.Split(string(data), "\n")
	inRequire := false

	// 用于去重
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// 解析 module 声明（当前项目）
		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			packages = append(packages, scanner.Package{
				Name:        moduleName,
				Version:     "", // module 没有版本
				PackageType: scanner.PackageTypeGolang,
				Source:      "module",
			})
			continue
		}

		// 解析 go 版本声明
		if strings.HasPrefix(line, "go ") {
			continue
		}

		// 解析 require 块开始
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}

		// 解析 require 块结束
		if line == ")" && inRequire {
			inRequire = false
			continue
		}

		// 解析 require 块内的依赖
		if inRequire {
			if pkg := parseRequireLine(line, seen); pkg != nil {
				packages = append(packages, *pkg)
			}
			continue
		}

		// 解析单行 require 格式: require github.com/foo/bar v1.0.0
		if strings.HasPrefix(line, "require ") {
			requireLine := strings.TrimPrefix(line, "require ")
			if pkg := parseRequireLine(requireLine, seen); pkg != nil {
				packages = append(packages, *pkg)
			}
			continue
		}

		// 解析 replace 指令
		if strings.HasPrefix(line, "replace ") {
			// replace github.com/foo/bar => github.com/bar/baz v1.0.0
			// 可以选择解析 replace 的目标，这里暂不处理
			continue
		}

		// 解析 exclude 指令
		if strings.HasPrefix(line, "exclude ") {
			continue
		}
	}

	return packages, nil
}

// parseRequireLine 解析单行 require 格式
func parseRequireLine(line string, seen map[string]bool) *scanner.Package {
	// 跳过注释（indirect）
	if strings.HasSuffix(line, "// indirect") {
		line = strings.TrimSuffix(line, "// indirect")
		line = strings.TrimSpace(line)
	}

	parts := strings.Fields(line)
	if len(parts) >= 2 {
		name := parts[0]
		version := strings.Trim(parts[1], " ()")

		// 去重
		key := name + "@" + version
		if seen[key] {
			return nil
		}
		seen[key] = true

		return &scanner.Package{
			Name:        name,
			Version:     version,
			PackageType: scanner.PackageTypeGolang,
		}
	}
	return nil
}

// parseRequirementsTxt 解析 requirements.txt 文件
func parseRequirementsTxt(path string) ([]scanner.Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packages []scanner.Package
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "==", 2)
		if len(parts) < 2 {
			parts = strings.SplitN(line, ">=", 2)
		}
		if len(parts) >= 2 {
			packages = append(packages, scanner.Package{
				Name:        parts[0],
				Version:     strings.Trim(parts[1], " <>=!"),
				PackageType: scanner.PackageTypePypi,
			})
		}
	}

	return packages, nil
}

// parseCargoToml 解析 Cargo.toml 文件
func parseCargoToml(path string) ([]scanner.Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest struct {
		Package *struct {
			Name    string `toml:"name"`
			Version string `toml:"version"`
		} `toml:"package"`
		Dependencies map[string]interface{} `toml:"dependencies"`
	}

	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	var packages []scanner.Package

	if manifest.Package != nil {
		packages = append(packages, scanner.Package{
			Name:        manifest.Package.Name,
			Version:     manifest.Package.Version,
			PackageType: scanner.PackageTypeCargo,
		})
	}

	for name, dep := range manifest.Dependencies {
		var version string
		switch v := dep.(type) {
		case string:
			version = v
		case map[string]interface{}:
			if vs, ok := v["version"].(string); ok {
				version = vs
			}
		}
		packages = append(packages, scanner.Package{
			Name:        name,
			Version:     version,
			PackageType: scanner.PackageTypeCargo,
		})
	}

	return packages, nil
}

// parsePackageJson 解析 package.json 文件
func parsePackageJson(path string) ([]scanner.Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Name         string            `json:"name"`
		Version      string            `json:"version"`
		Dependencies map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	var packages []scanner.Package

	for name, version := range pkg.Dependencies {
		packages = append(packages, scanner.Package{
			Name:        name,
			Version:     strings.Trim(version, "^~>= "),
			PackageType: scanner.PackageTypeNpm,
		})
	}

	return packages, nil
}

// syncCommand 返回同步命令
func syncCommand() *cli.Command {
	return &cli.Command{
		Name:      "sync",
		Aliases:   []string{"y"},
		Usage:     "Sync scanned packages to TechStack service",
		UsageText: "techstacktools sync [options]",
		Description: `Sync packages JSON file to TechStack service.
Use scan command with --export to generate JSON file, then sync to service.
Example: techstacktools scan -e packages.json && techstacktools sync -s http://localhost:8080 --path packages.json`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "server",
				Aliases:     []string{"s"},
				Usage:       "TechStack service URL",
				DefaultText: os.Getenv("TECHSTACK_SERVER"),
			},
			&cli.StringFlag{
				Name:        "ak",
				Aliases:     []string{"a"},
				Usage:       "Access Key",
				DefaultText: os.Getenv("TECHSTACK_AK"),
			},
			&cli.StringFlag{
				Name:        "sk",
				Usage:       "Secret Key",
				DefaultText: os.Getenv("TECHSTACK_SK"),
			},
			&cli.BoolFlag{
				Name:    "batch",
				Aliases: []string{"b"},
				Usage:   "Batch import mode",
			},
			&cli.BoolFlag{
				Name:    "skipError",
				Aliases: []string{"k"},
				Usage:   "Skip errors and continue",
			},
			&cli.StringFlag{
				Name:    "path",
				Aliases: []string{"p"},
				Usage:   "Import from JSON file",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Detailed output",
			},
		},
		Action: runSync,
	}
}

// runSync 执行同步
func runSync(ctx context.Context, cmd *cli.Command) error {
	// 获取选项
	server := cmd.String("server")
	ak := cmd.String("ak")
	sk := cmd.String("sk")
	batch := cmd.Bool("batch")
	skipError := cmd.Bool("skipError")
	importFile := cmd.String("path")
	verbose := cmd.Bool("verbose")

	// 从环境变量获取默认值
	if server == "" {
		server = os.Getenv("TECHSTACK_SERVER")
	}
	if ak == "" {
		ak = os.Getenv("TECHSTACK_AK")
	}
	if sk == "" {
		sk = os.Getenv("TECHSTACK_SK")
	}

	// 验证必要参数
	if server == "" || ak == "" || sk == "" {
		fmt.Println("Error: --server (or TECHSTACK_SERVER), --ak (or TECHSTACK_AK), --sk (or TECHSTACK_SK) are required")
		os.Exit(1)
	}

	// 检查导入文件
	if importFile == "" {
		fmt.Println("Error: --path is required to specify JSON file")
		fmt.Println("Example: techstacktools scan -e packages.json && techstacktools sync --path packages.json")
		os.Exit(1)
	}

	// 读取 JSON 文件
	data, err := os.ReadFile(importFile)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", importFile, err)
	}

	var packages []scanner.Package
	if err := json.Unmarshal(data, &packages); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	if verbose {
		fmt.Printf("Read %d packages from %s\n", len(packages), importFile)
	}

	// 转换包格式
	var packagesToSync []map[string]string
	for _, pkg := range packages {
		packagesToSync = append(packagesToSync, map[string]string{
			"name":      pkg.Name,
			"purl_type": string(pkg.PackageType),
			"version":   pkg.Version,
		})
	}

	// 创建客户端
	client := sync.NewClientWithAKSK(server, ak, sk)
	if verbose {
		fmt.Printf("Syncing to %s...\n", server)
	}

	// 执行同步
	if batch {
		if verbose {
			fmt.Printf("Using batch import mode (max %d per batch)\n", sync.BatchSize)
		}
		if err := client.BatchImportLibraries(packagesToSync); err != nil {
			if skipError {
				if verbose {
					fmt.Printf("Batch import completed with errors: %v\n", err)
				}
			} else {
				return fmt.Errorf("batch import failed: %w", err)
			}
		}
	} else {
		if err := client.AddLibraries(packagesToSync); err != nil {
			if skipError {
				if verbose {
					fmt.Printf("Sync completed with errors: %v\n", err)
				}
			} else {
				return fmt.Errorf("sync failed: %w", err)
			}
		}
	}

	if verbose {
		fmt.Printf("Successfully synced %d packages\n", len(packagesToSync))
	} else {
		fmt.Printf("Synced %d packages\n", len(packagesToSync))
	}

	return nil
}

// importCommand 返回导入命令
func importCommand() *cli.Command {
	return &cli.Command{
		Name:      "import",
		Aliases:   []string{"i"},
		Usage:     "Import package data from external sources",
		UsageText: "techstacktools import <subcommand> [options]",
		Description: `Import package data from various external sources into TechStack database.
Supported subcommands:
  cargo <path/to/crates.csv>  - Import cargo packages from crates.io CSV
  golang [path/to/duckdb.db]  - Import Go modules from DuckDB database`,
		Commands: []*cli.Command{
			{
				Name:    "crate",
				Aliases: []string{"rust", "cargo"},
				Usage:   "Import cargo packages from crates.io (CSV or DuckDB)",
				UsageText: `techstacktools import cargo [options] [path]
Options:
  --source duckdb   Import from DuckDB database (default: csv)
  --source csv      Import from CSV file
Examples:
  techstacktools import cargo --source csv crates.csv
  techstacktools import cargo --source duckdb --config ../config.yml crates_io.duckdb
  techstacktools import cargo --source duckdb  # uses crates_io.duckdb by default`,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "source",
						Usage: "Data source type: csv or duckdb",
						Value: "csv",
					},
					&cli.StringFlag{
						Name:  "config",
						Usage: "Configuration file path",
						Value: "config.yml",
					},
				},
				Action: tools.ImportCrates,
			},
			{
				Name:      "golang",
				Aliases:   []string{"go", "mod"},
				Usage:     "Import Go modules from DuckDB database",
				UsageText: "techstacktools import golang [path/to/go_index.duckdb]",
				Action:    tools.ImportGolangModules,
			},
		},
	}
}

// parseTypes 解析逗号分隔的类型列表
func parseTypes(typeStr string) []string {
	typeStr = strings.ToLower(typeStr)

	if typeStr == "all" {
		return []string{"golang", "npm", "pypi", "cargo"}
	}

	types := strings.Split(typeStr, ",")
	var result []string
	for _, t := range types {
		t = strings.TrimSpace(t)
		if t != "" {
			result = append(result, t)
		}
	}

	return result
}

// createScanners 根据类型创建扫描器
func createScanners(types []string, path string) ([]scanner.Scanner, error) {
	var scanners []scanner.Scanner

	for _, t := range types {
		switch t {
		case "golang":
			s := scanner.NewGolangScanner()
			scanners = append(scanners, s)
		case "npm":
			scanners = append(scanners, scanner.NewNpmScanner())
		case "pypi":
			scanners = append(scanners, scanner.NewPypiScanner())
		case "cargo":
			cargoScanner := scanner.NewCargoScanner(path).
				SetRecursive(scanFlags.Recursive).
				SetIncludeDev(scanFlags.IncludeDev).
				SetIncludeBuild(scanFlags.IncludeBuild).
				SetAnalyzeGraph(scanFlags.GraphAnalysis)
			scanners = append(scanners, cargoScanner)
		}
	}

	return scanners, nil
}

// deduplicateByPURL 按 PURL 去重
func deduplicateByPURL(packages []scanner.Package) []scanner.Package {
	seen := make(map[string]scanner.Package)
	for _, pkg := range packages {
		purl := fmt.Sprintf("%s:%s", pkg.PackageType, pkg.Name)
		if _, ok := seen[purl]; !ok {
			seen[purl] = pkg
		}
	}

	var result []scanner.Package
	for _, pkg := range seen {
		result = append(result, pkg)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].PackageType != result[j].PackageType {
			return result[i].PackageType < result[j].PackageType
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// printPackages 打印包列表
func printPackages(packages []scanner.Package, format string) {
	switch format {
	case "text":
		for _, pkg := range packages {
			fmt.Printf("%s:%s@%s\n", pkg.PackageType, pkg.Name, pkg.Version)
		}
	case "table":
		fmt.Printf("%-40s %-15s %-10s\n", "NAME", "VERSION", "TYPE")
		fmt.Printf("%s\n", strings.Repeat("-", 70))
		for _, pkg := range packages {
			fmt.Printf("%-40s %-15s %-10s\n", pkg.Name, pkg.Version, pkg.PackageType)
		}
	default:
		data, err := json.MarshalIndent(packages, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling packages: %v\n", err)
			return
		}
		fmt.Println(string(data))
	}
}

// exportPackages 导出包到文件
func exportPackages(packages []scanner.Package, filePath string, format string) {
	var data []byte
	var err error

	switch format {
	case "text":
		var lines []string
		for _, pkg := range packages {
			lines = append(lines, fmt.Sprintf("%s:%s@%s", pkg.PackageType, pkg.Name, pkg.Version))
		}
		data = []byte(strings.Join(lines, "\n"))
	case "json":
		data, err = json.MarshalIndent(packages, "", "  ")
	default:
		data, err = json.MarshalIndent(packages, "", "  ")
	}

	if err != nil {
		fmt.Printf("Error marshaling packages: %v\n", err)
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Printf("Exported %d packages to %s\n", len(packages), filePath)
}

// 调用外部命令检查 cargo tree
func checkCargoTree(path string) (string, error) {
	cmd := exec.Command("cargo", "tree", "--format", "{p}")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
