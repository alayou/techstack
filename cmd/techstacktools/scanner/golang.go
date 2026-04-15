package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GolangScanner Go包扫描器
type GolangScanner struct {
	gopath string
}

// NewGolangScanner 创建Go包扫描器
func NewGolangScanner() *GolangScanner {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// 默认 GOPATH
		home := os.Getenv("HOME")
		gopath = filepath.Join(home, "go")
	}
	return &GolangScanner{gopath: gopath}
}

// SetGopath 设置自定义 GOPATH
func (s *GolangScanner) SetGopath(gopath string) *GolangScanner {
	if gopath != "" {
		s.gopath = gopath
	}
	return s
}

// Name 返回扫描器名称
func (s *GolangScanner) Name() string {
	return "golang"
}

// Type 返回包类型
func (s *GolangScanner) Type() PackageType {
	return PackageTypeGolang
}

// Scan 扫描GOPATH/pkg/mod目录下的Go包
func (s *GolangScanner) Scan() ([]Package, error) {
	modPath := filepath.Join(s.gopath, "pkg", "mod")

	// 检查目录是否存在
	info, err := os.Stat(modPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 目录不存在，返回空列表
			return []Package{}, nil
		}
		return nil, fmt.Errorf("stat mod path: %w", err)
	}

	if !info.IsDir() {
		return []Package{}, nil
	}

	var packages []Package

	// 遍历 mod 目录
	entries, err := os.ReadDir(modPath)
	if err != nil {
		return nil, fmt.Errorf("read mod directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 跳过缓存目录
		if entry.Name() == "cache" {
			continue
		}

		// 解析目录名：可能是 module@version 或 module/version@version 格式
		modulePath := filepath.Join(modPath, entry.Name())
		packages = append(packages, s.scanModuleDir(entry.Name(), modulePath)...)
	}

	return packages, nil
}

// scanModuleDir 扫描单个模块目录
func (s *GolangScanner) scanModuleDir(dirName, dirPath string) []Package {
	var packages []Package

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return packages
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 解析包名和版本
		// 目录名格式：module@version 或 module/version@version
		packageName, version := s.parseDirName(entry.Name())
		if packageName == "" {
			continue
		}

		// 组合完整包名
		fullName := dirName + "/" + packageName
		if version == "" {
			packages = append(packages, s.scanModuleDir(fullName, filepath.Join(dirPath, entry.Name()))...)
			// fmt.Println(strings.Repeat("=", 30), fullName, filepath.Join(dirPath, entry.Name()))
			continue
		}
		packages = append(packages, Package{
			Name:        fullName,
			Version:     version,
			PackageType: PackageTypeGolang,
			InstallPath: filepath.Join(dirPath, entry.Name()),
		})
	}

	return packages
}

// parseDirName 解析目录名获取包名和版本
func (s *GolangScanner) parseDirName(dirName string) (name, version string) {
	// 格式1: module@version
	// 格式2: module/version@version
	// 格式3: module (没有版本)

	// 查找 @ 分隔符
	atIndex := strings.LastIndex(dirName, "@")
	if atIndex == -1 {
		// 没有版本号
		return dirName, ""
	}

	name = dirName[:atIndex]
	version = dirName[atIndex+1:]

	// 处理带有斜杠的情况
	if strings.Contains(name, "/") {
		// 已经包含路径，直接返回
		return name, version
	}

	return name, version
}
