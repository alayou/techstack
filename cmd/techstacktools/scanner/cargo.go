package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// CargoManifest represents a Cargo.toml file structure
type CargoManifest struct {
	Package           *CargoPackage          `toml:"package"`
	Lib               *CargoLib              `toml:"lib"`
	Dependencies      map[string]interface{} `toml:"dependencies"`
	DevDependencies   map[string]interface{} `toml:"dev-dependencies"`
	BuildDependencies map[string]interface{} `toml:"build-dependencies"`
	Workspace         *CargoWorkspace        `toml:"workspace"`
	Features          map[string][]string    `toml:"features"`
}

// CargoPackage represents the [package] section
type CargoPackage struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Edition     string   `toml:"edition"`
	Description string   `toml:"description"`
	Homepage    string   `toml:"homepage"`
	Repository  string   `toml:"repository"`
	License     string   `toml:"license"`
	Keywords    []string `toml:"keywords"`
	Authors     []string `toml:"authors"`
	Categories  []string `toml:"categories"`
}

// CargoLib represents the [lib] section
type CargoLib struct {
	Name string `toml:"name"`
	Path string `toml:"path"`
}

// CargoWorkspace represents the [workspace] section
type CargoWorkspace struct {
	Members     []string `toml:"members"`
	Exclude     []string `toml:"exclude"`
	RootPackage string   `toml:"root"`
}

// CargoDependency represents a dependency entry
type CargoDependency struct {
	Name            string
	Version         string
	Registry        string
	Git             string
	Branch          string
	Tag             string
	Rev             string
	Path            string
	Optional        bool
	Features        []string
	DefaultFeatures bool
	Package         string
}

// DependencyGraph represents the dependency graph of a Cargo project
type DependencyGraph struct {
	RootPackage    string
	Dependencies   map[string]*CargoDependency
	TransitiveDeps map[string]map[string]*CargoDependency
}

// CargoScanner Cargo.toml 扫描器
type CargoScanner struct {
	rootPath     string
	recursive    bool
	includeDev   bool
	includeBuild bool
	analyzeGraph bool
}

// NewCargoScanner 创建 Cargo 扫描器
func NewCargoScanner(rootPath string) *CargoScanner {
	return &CargoScanner{
		rootPath:     rootPath,
		recursive:    true,
		includeDev:   true,
		includeBuild: true,
		analyzeGraph: true,
	}
}

// Name 返回扫描器名称
func (s *CargoScanner) Name() string {
	return "cargo"
}

// Type 返回包类型
func (s *CargoScanner) Type() PackageType {
	return PackageTypeCargo
}

// SetRecursive 设置是否递归扫描子目录
func (s *CargoScanner) SetRecursive(recursive bool) *CargoScanner {
	s.recursive = recursive
	return s
}

// SetIncludeDev 设置是否包含开发依赖
func (s *CargoScanner) SetIncludeDev(include bool) *CargoScanner {
	s.includeDev = include
	return s
}

// SetIncludeBuild 设置是否包含构建依赖
func (s *CargoScanner) SetIncludeBuild(include bool) *CargoScanner {
	s.includeBuild = include
	return s
}

// SetAnalyzeGraph 设置是否分析依赖图
func (s *CargoScanner) SetAnalyzeGraph(analyze bool) *CargoScanner {
	s.analyzeGraph = analyze
	return s
}

// GetRootPath 获取扫描根路径
func (s *CargoScanner) GetRootPath() string {
	return s.rootPath
}

// Scan 扫描目录中的 Cargo.toml 文件
func (s *CargoScanner) Scan() ([]Package, error) {
	var packages []Package

	// 解析 root path
	absPath, err := filepath.Abs(s.rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// 检查是否是 git 仓库
	if isGitRepo(absPath) {
		// 如果是 git 仓库，先克隆或更新
		packages = append(packages, s.scanGitRepo(absPath)...)
	} else {
		// 扫描本地目录
		packages = append(packages, s.scanDirectory(absPath)...)
	}

	return packages, nil
}

// scanDirectory 扫描目录
func (s *CargoScanner) scanDirectory(dirPath string) []Package {
	var packages []Package

	info, err := os.Stat(dirPath)
	if err != nil {
		return packages
	}

	if info.IsDir() {
		// 查找 Cargo.toml 文件
		tomlPath := filepath.Join(dirPath, "Cargo.toml")
		if _, err := os.Stat(tomlPath); err == nil {
			pkgs := s.parseCargoToml(tomlPath)
			packages = append(packages, pkgs...)
		}

		// 递归扫描子目录
		if s.recursive {
			entries, err := os.ReadDir(dirPath)
			if err != nil {
				return packages
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				// 跳过隐藏目录和常见忽略目录
				name := entry.Name()
				if strings.HasPrefix(name, ".") ||
					name == "target" ||
					name == "node_modules" ||
					name == "__pycache__" ||
					name == "venv" ||
					name == ".venv" {
					continue
				}

				subPath := filepath.Join(dirPath, name)
				packages = append(packages, s.scanDirectory(subPath)...)
			}
		}
	}

	return packages
}

// scanGitRepo 扫描 git 仓库
func (s *CargoScanner) scanGitRepo(repoPath string) []Package {
	var packages []Package

	// 检查是否是 git 仓库
	if !isGitRepo(repoPath) {
		// 不是 git 仓库，当作普通目录处理
		return s.scanDirectory(repoPath)
	}

	// 获取仓库根目录
	rootPath, err := getGitRoot(repoPath)
	if err != nil {
		// 如果获取失败，当作普通目录处理
		return s.scanDirectory(repoPath)
	}

	// 扫描仓库
	packages = append(packages, s.scanDirectory(rootPath)...)

	return packages
}

// parseCargoToml 解析 Cargo.toml 文件
func (s *CargoScanner) parseCargoToml(tomlPath string) []Package {
	var packages []Package

	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return packages
	}

	var manifest CargoManifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return packages
	}

	// 获取 Cargo.toml 所在目录
	dirPath := filepath.Dir(tomlPath)

	// 解析主包信息
	if manifest.Package != nil {
		pkg := Package{
			Name:        manifest.Package.Name,
			Version:     manifest.Package.Version,
			PackageType: PackageTypeCargo,
			PackagePath: dirPath,
		}
		packages = append(packages, pkg)
	}

	// 解析生产依赖
	if manifest.Dependencies != nil {
		for name, dep := range manifest.Dependencies {
			depInfo := s.parseDependencyValue(name, dep, ScopeProd, dirPath)
			packages = append(packages, depInfo)
		}
	}

	// 解析开发依赖
	if s.includeDev && manifest.DevDependencies != nil {
		for name, dep := range manifest.DevDependencies {
			depInfo := s.parseDependencyValue(name, dep, ScopeDev, dirPath)
			packages = append(packages, depInfo)
		}
	}

	// 解析构建依赖
	if s.includeBuild && manifest.BuildDependencies != nil {
		for name, dep := range manifest.BuildDependencies {
			depInfo := s.parseDependencyValue(name, dep, ScopeBuild, dirPath)
			packages = append(packages, depInfo)
		}
	}

	return packages
}

// parseDependencyValue 解析依赖值
func (s *CargoScanner) parseDependencyValue(name string, depValue interface{}, scope DependencyScope, dirPath string) Package {
	pkg := Package{
		Name:        name,
		PackageType: PackageTypeCargo,
		Scope:       scope,
		PackagePath: dirPath,
	}

	switch v := depValue.(type) {
	case string:
		// 简单字符串形式: "1.0.0" 或 "^1.0.0"
		pkg.Version = v

	case map[string]interface{}:
		// 复杂形式: { version = "1.0.0", features = [...] }
		if version, ok := v["version"].(string); ok {
			pkg.Version = version
		}
		if registry, ok := v["registry"].(string); ok {
			pkg.Registry = registry
		}
		if git, ok := v["git"].(string); ok {
			pkg.Source = "git+" + git
			if branch, ok := v["branch"].(string); ok {
				pkg.Source += "?branch=" + branch
			} else if tag, ok := v["tag"].(string); ok {
				pkg.Source += "?tag=" + tag
			} else if rev, ok := v["rev"].(string); ok {
				pkg.Source += "?rev=" + rev
			}
		}
		if path, ok := v["path"].(string); ok {
			pkg.Source = "path+" + filepath.Join(dirPath, path)
		}
		if features, ok := v["features"].([]interface{}); ok {
			for _, f := range features {
				if fs, ok := f.(string); ok {
					pkg.FeatureFlags = append(pkg.FeatureFlags, fs)
				}
			}
		}

	default:
		// 尝试转换为字符串
		pkg.Version = fmt.Sprintf("%v", v)
	}

	return pkg
}

// BuildDependencyGraph 构建依赖图
func (s *CargoScanner) BuildDependencyGraph(rootPath string) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Dependencies:   make(map[string]*CargoDependency),
		TransitiveDeps: make(map[string]map[string]*CargoDependency),
	}

	// 首先解析根 Cargo.toml
	tomlPath := filepath.Join(rootPath, "Cargo.toml")
	if _, err := os.Stat(tomlPath); err != nil {
		return graph, fmt.Errorf("Cargo.toml not found: %w", err)
	}

	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return graph, fmt.Errorf("failed to read Cargo.toml: %w", err)
	}

	var manifest CargoManifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return graph, fmt.Errorf("failed to parse Cargo.toml: %w", err)
	}

	// 设置根包名
	if manifest.Package != nil {
		graph.RootPackage = manifest.Package.Name
	}

	// 收集直接依赖
	s.collectDirectDependencies(manifest, graph)

	// 分析传递依赖（如果启用）
	if s.analyzeGraph {
		if err := s.analyzeTransitiveDependencies(rootPath, graph); err != nil {
			fmt.Printf("Warning: failed to analyze transitive dependencies: %v\n", err)
		}
	}

	return graph, nil
}

// collectDirectDependencies 收集直接依赖
func (s *CargoScanner) collectDirectDependencies(manifest CargoManifest, graph *DependencyGraph) {
	// 生产依赖
	if manifest.Dependencies != nil {
		for name, dep := range manifest.Dependencies {
			depInfo := extractDependencyInfo(name, dep)
			graph.Dependencies[name] = depInfo
		}
	}

	// 开发依赖
	if s.includeDev && manifest.DevDependencies != nil {
		for name, dep := range manifest.DevDependencies {
			depInfo := extractDependencyInfo(name, dep)
			graph.Dependencies[name] = depInfo
		}
	}

	// 构建依赖
	if s.includeBuild && manifest.BuildDependencies != nil {
		for name, dep := range manifest.BuildDependencies {
			depInfo := extractDependencyInfo(name, dep)
			graph.Dependencies[name] = depInfo
		}
	}
}

// extractDependencyInfo 提取依赖信息
func extractDependencyInfo(name string, dep interface{}) *CargoDependency {
	depInfo := &CargoDependency{
		Name:            name,
		DefaultFeatures: true,
	}

	switch v := dep.(type) {
	case string:
		depInfo.Version = v

	case map[string]interface{}:
		if version, ok := v["version"].(string); ok {
			depInfo.Version = version
		}
		if registry, ok := v["registry"].(string); ok {
			depInfo.Registry = registry
		}
		if git, ok := v["git"].(string); ok {
			depInfo.Git = git
		}
		if branch, ok := v["branch"].(string); ok {
			depInfo.Branch = branch
		}
		if tag, ok := v["tag"].(string); ok {
			depInfo.Tag = tag
		}
		if rev, ok := v["rev"].(string); ok {
			depInfo.Rev = rev
		}
		if path, ok := v["path"].(string); ok {
			depInfo.Path = path
		}
		if optional, ok := v["optional"].(bool); ok {
			depInfo.Optional = optional
		}
		if features, ok := v["features"].([]interface{}); ok {
			for _, f := range features {
				if fs, ok := f.(string); ok {
					depInfo.Features = append(depInfo.Features, fs)
				}
			}
		}
		if df, ok := v["default-features"].(bool); ok {
			depInfo.DefaultFeatures = df
		}
	}

	return depInfo
}

// analyzeTransitiveDependencies 分析传递依赖
func (s *CargoScanner) analyzeTransitiveDependencies(rootPath string, graph *DependencyGraph) error {
	// 尝试使用 cargo tree 命令获取依赖树
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo not found in PATH")
	}

	// 运行 cargo tree
	cmd := exec.Command("cargo", "tree", "--format", "{p}{f}")
	cmd.Dir = rootPath

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run cargo tree: %w", err)
	}

	// 解析 cargo tree 输出
	lines := strings.Split(string(output), "\n")
	currentPkg := graph.RootPackage

	if currentPkg != "" {
		graph.TransitiveDeps[currentPkg] = make(map[string]*CargoDependency)
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析依赖行
		// 格式: crate_name v1.0.0 -> dep1 v1.0.0, dep2 v2.0.0
		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			continue
		}

		pkgName := parts[0]
		// 跳过根包
		if pkgName == currentPkg {
			continue
		}

		// 解析版本
		version := strings.Trim(parts[1], "()")

		// 提取特性标志
		var features []string
		if len(parts) > 2 {
			featurePart := strings.Join(parts[2:], "")
			featureMatch := regexp.MustCompile(`\[([^\]]+)\]`).FindStringSubmatch(featurePart)
			if featureMatch != nil {
				features = strings.Split(featureMatch[1], ",")
			}
		}

		graph.TransitiveDeps[currentPkg][pkgName] = &CargoDependency{
			Name:     pkgName,
			Version:  version,
			Features: features,
		}
	}

	return nil
}

// isGitRepo 检查目录是否是 git 仓库
func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err == nil {
		return info.IsDir() || info.Name() == ".git"
	}

	// 检查父目录
	parent := filepath.Dir(path)
	if parent != path && parent != "." {
		return isGitRepo(parent)
	}

	return false
}

// getGitRoot 获取 git 仓库根目录
func getGitRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return path, err
	}
	return strings.TrimSpace(string(output)), nil
}

// CloneGitRepo 克隆 git 仓库到本地
func CloneGitRepo(repoURL, targetDir string) (string, error) {
	// 检查目标目录是否存在
	if _, err := os.Stat(targetDir); err == nil {
		// 目录已存在，执行 git pull
		cmd := exec.Command("git", "pull")
		cmd.Dir = targetDir
		if err := cmd.Run(); err != nil {
			return targetDir, fmt.Errorf("failed to pull repository: %w", err)
		}
		return targetDir, nil
	}

	// 克隆仓库
	cmd := exec.Command("git", "clone", repoURL, targetDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	return targetDir, nil
}
