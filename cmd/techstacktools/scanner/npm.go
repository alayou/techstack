package scanner

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NpmScanner npm包扫描器
type NpmScanner struct {
	rootPath string
}

// NewNpmScanner 创建npm包扫描器
func NewNpmScanner() *NpmScanner {
	// 尝试获取全局npm root
	rootPath := ""
	cmd := exec.Command("npm", "root", "-g")
	out, err := cmd.Output()
	if err == nil {
		rootPath = strings.TrimSpace(string(out))
	}
	return &NpmScanner{rootPath: rootPath}
}

// Name 返回扫描器名称
func (s *NpmScanner) Name() string {
	return "npm"
}

// Type 返回包类型
func (s *NpmScanner) Type() PackageType {
	return PackageTypeNpm
}

// Scan 扫描node_modules目录下的npm包
func (s *NpmScanner) Scan() ([]Package, error) {
	var packages []Package

	// 1. 扫描当前目录向上查找node_modules
	packages = append(packages, s.scanUpward()...)

	// 2. 扫描全局目录
	if s.rootPath != "" {
		info, err := os.Stat(s.rootPath)
		if err == nil && info.IsDir() {
			packages = append(packages, s.scanNodeModules(s.rootPath)...)
		}
	}

	return packages, nil
}

// scanUpward 向上查找node_modules目录
func (s *NpmScanner) scanUpward() []Package {
	var packages []Package

	// 从当前工作目录向上查找
	cwd, err := os.Getwd()
	if err != nil {
		return packages
	}

	// 向上查找最多5层
	for i := 0; i < 5; i++ {
		nodeModulesPath := filepath.Join(cwd, "node_modules")
		info, err := os.Stat(nodeModulesPath)
		if err == nil && info.IsDir() {
			packages = append(packages, s.scanNodeModules(nodeModulesPath)...)
			break
		}

		// 上一级目录
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return packages
}

// scanNodeModules 扫描node_modules目录
func (s *NpmScanner) scanNodeModules(nodeModulesPath string) []Package {
	var packages []Package

	entries, err := os.ReadDir(nodeModulesPath)
	if err != nil {
		return packages
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 跳过特殊目录
		if entry.Name() == ".bin" || entry.Name() == ".package-lock.json" {
			continue
		}

		// 读取package.json
		pkgPath := filepath.Join(nodeModulesPath, entry.Name())
		pkgJSONPath := filepath.Join(pkgPath, "package.json")

		pkg := s.parsePackageJSON(pkgJSONPath)
		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}

	return packages
}

// parsePackageJSON 解析package.json文件
func (s *NpmScanner) parsePackageJSON(jsonPath string) Package {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return Package{}
	}

	var pkgInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	if err := json.Unmarshal(data, &pkgInfo); err != nil {
		return Package{}
	}

	if pkgInfo.Name == "" {
		return Package{}
	}

	return Package{
		Name:        pkgInfo.Name,
		Version:     pkgInfo.Version,
		PackageType: PackageTypeNpm,
		InstallPath: filepath.Dir(jsonPath),
	}
}
