package libparse

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// StandardizedPackage 标准化包信息
type StandardizedPackage struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Ecosystem  string `json:"ecosystem"`
	PURL       string `json:"purl"`
	SourceFile string `json:"sourceFile"`
	DepType    string `json:"depType"` // direct, dev, build, optional
}

// InventoryResult 扫描结果
type InventoryResult struct {
	Packages []StandardizedPackage `json:"packages"`
	Summary  *InventorySummary     `json:"summary"`
}

// InventorySummary 扫描摘要
type InventorySummary struct {
	TotalPackages int            `json:"totalPackages"`
	ByEcosystem   map[string]int `json:"byEcosystem"`
	ScanRoot      string         `json:"scanRoot"`
	Status        string         `json:"status"`
	Error         string         `json:"error,omitempty"`
}

// Scanner 依赖扫描器
// 使用现有 libparse 解析器扫描项目依赖
type Scanner struct{}

// NewScanner 创建新的扫描器实例
func NewScanner() *Scanner {
	return &Scanner{}
}

// ScanPath 扫描本地目录
func (s *Scanner) ScanPath(ctx context.Context, path string) (*InventoryResult, error) {
	// 验证路径存在
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", path)
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	result := &InventoryResult{
		Packages: make([]StandardizedPackage, 0),
		Summary: &InventorySummary{
			ByEcosystem: make(map[string]int),
			ScanRoot:    absPath,
		},
	}

	// 扫描依赖文件
	err = scanDependencyFiles(absPath, result)
	if err != nil {
		result.Summary.Status = "partial"
		result.Summary.Error = err.Error()
	} else {
		result.Summary.Status = "success"
	}

	result.Summary.TotalPackages = len(result.Packages)
	return result, nil
}

// scanDependencyFiles 扫描项目中的依赖文件
func scanDependencyFiles(root string, result *InventoryResult) error {
	// 定义依赖文件解析配置
	type parserConfig struct {
		files []string
		isDev bool
	}

	parsers := []struct {
		files []string
		isDev bool
	}{
		{files: []string{"go.mod"}, isDev: false},
		{files: []string{"package.json"}, isDev: false},
		{files: []string{"pyproject.toml"}, isDev: false},
		{files: []string{"requirements.txt"}, isDev: false},
		{files: []string{"Cargo.toml"}, isDev: false},
	}

	// 遍历解析器
	for _, fp := range parsers {
		for _, file := range fp.files {
			path := filepath.Join(root, file)
			if _, err := os.Stat(path); err == nil {
				pkgs, err := parseFile(path, file)
				if err != nil {
					continue
				}
				for _, pkg := range pkgs {
					depType := "direct"
					if fp.isDev {
						depType = "dev"
					}
					result.Packages = append(result.Packages, StandardizedPackage{
						Name:       pkg.Name,
						Version:    pkg.Version,
						Ecosystem:  pkg.Ecosystem,
						SourceFile: file,
						DepType:    depType,
						PURL:       generatePURL(pkg.Ecosystem, pkg.Name, pkg.Version),
					})
					result.Summary.ByEcosystem[pkg.Ecosystem]++
				}
			}
		}
	}

	return nil
}

// parseFile 根据文件名解析依赖文件
func parseFile(path, filename string) ([]PackageInfo, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	switch filename {
	case "go.mod":
		return ParseGoMod(string(content))
	case "package.json":
		return ParsePackageJson(string(content))
	case "pyproject.toml":
		return ParsePyproject(string(content))
	case "requirements.txt":
		return ParseRequirementsTxt(string(content))
	case "Cargo.toml":
		return ParseCargoToml(string(content))
	default:
		return nil, nil
	}
}

// ScanGitRepo 扫描 Git 仓库
func (s *Scanner) ScanGitRepo(ctx context.Context, repoURL string) (*InventoryResult, error) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "scalibr-scan-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 检查 git 是否可用
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git not found: %w", err)
	}

	// 克隆仓库
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, tmpDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone repo: %w", err)
	}

	// 扫描克隆的目录
	return s.ScanPath(ctx, tmpDir)
}

// ScanInput 扫描输入（支持本地路径或 Git 仓库地址）
func (s *Scanner) ScanInput(ctx context.Context, input string) (*InventoryResult, error) {
	// 判断是否为 Git 仓库
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "git@") || strings.HasSuffix(input, ".git") {
		return s.ScanGitRepo(ctx, input)
	}

	// 本地路径
	return s.ScanPath(ctx, input)
}

// generatePURL 根据生态系统生成 PURL
func generatePURL(ecosystem, name, version string) string {
	// 简单的 PURL 生成
	switch ecosystem {
	case "pypi":
		return fmt.Sprintf("pkg:pypi/%s@%s", name, version)
	case "npm":
		return fmt.Sprintf("pkg:npm/%s@%s", name, version)
	case "go":
		return fmt.Sprintf("pkg:golang/%s@%s", name, version)
	case "cargo":
		return fmt.Sprintf("pkg:cargo/%s@%s", name, version)
	default:
		if version != "" {
			return fmt.Sprintf("pkg:generic/%s@%s", name, version)
		}
		return fmt.Sprintf("pkg:generic/%s", name)
	}
}
