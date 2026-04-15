package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// PypiScanner PyPI包扫描器
type PypiScanner struct {
	sitePackages string
}

// NewPypiScanner 创建PyPI包扫描器
func NewPypiScanner() *PypiScanner {
	// 获取 site-packages 路径
	sitePackages := ""

	// 尝试使用 python -m site
	cmd := exec.Command("python3", "-m", "site", "--site-packages")
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "sys.path") {
				sitePackages = line
				break
			}
		}
	}

	// 如果上面的方法失败，尝试 python -c
	if sitePackages == "" {
		cmd = exec.Command("python3", "-c", "import site; print(site.getsitepackages()[0])")
		out, err = cmd.Output()
		if err == nil {
			sitePackages = strings.TrimSpace(string(out))
		}
	}

	return &PypiScanner{sitePackages: sitePackages}
}

// Name 返回扫描器名称
func (s *PypiScanner) Name() string {
	return "pypi"
}

// Type 返回包类型
func (s *PypiScanner) Type() PackageType {
	return PackageTypePypi
}

// Scan 扫描site-packages目录下的PyPI包
func (s *PypiScanner) Scan() ([]Package, error) {
	if s.sitePackages == "" {
		return []Package{}, fmt.Errorf("cannot find site-packages directory")
	}

	info, err := os.Stat(s.sitePackages)
	if err != nil {
		return nil, fmt.Errorf("stat site-packages: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("site-packages is not a directory")
	}

	return s.scanSitePackages(s.sitePackages), nil
}

// scanSitePackages 扫描site-packages目录
func (s *PypiScanner) scanSitePackages(sitePackagesPath string) []Package {
	var packages []Package

	entries, err := os.ReadDir(sitePackagesPath)
	if err != nil {
		return packages
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// 解析 DIST-INFO 目录
		if strings.HasSuffix(name, ".dist-info") {
			pkg := s.parseDistInfo(filepath.Join(sitePackagesPath, name))
			if pkg.Name != "" {
				packages = append(packages, pkg)
			}
			continue
		}

		// 解析 EGG-INFO 目录
		if strings.HasSuffix(name, ".egg-info") {
			pkg := s.parseEggInfo(filepath.Join(sitePackagesPath, name))
			if pkg.Name != "" {
				packages = append(packages, pkg)
			}
		}
	}

	return packages
}

// parseDistInfo 解析 .dist-info 目录
func (s *PypiScanner) parseDistInfo(distInfoPath string) Package {
	// 读取 METADATA 文件
	metadataPath := filepath.Join(distInfoPath, "METADATA")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return Package{}
	}

	// 解析 Metadata-Version 和 Name/Version
	lines := strings.Split(string(data), "\n")
	name := ""
	version := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		}
		if strings.HasPrefix(line, "Version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
	}

	if name == "" {
		return Package{}
	}

	// 从目录名解析
	dirName := filepath.Base(distInfoPath)
	re := regexp.MustCompile(`^(.+)-(.+)\.dist-info$`)
	matches := re.FindStringSubmatch(dirName)
	if len(matches) == 3 {
		name = matches[1]
		version = matches[2]
	}

	return Package{
		Name:        name,
		Version:     version,
		PackageType: PackageTypePypi,
		InstallPath: distInfoPath,
	}
}

// parseEggInfo 解析 .egg-info 目录
func (s *PypiScanner) parseEggInfo(eggInfoPath string) Package {
	// 读取 PKG-INFO 文件
	pkgInfoPath := filepath.Join(eggInfoPath, "PKG-INFO")
	data, err := os.ReadFile(pkgInfoPath)
	if err != nil {
		return Package{}
	}

	lines := strings.Split(string(data), "\n")
	name := ""
	version := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		}
		if strings.HasPrefix(line, "Version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
	}

	if name == "" {
		return Package{}
	}

	return Package{
		Name:        name,
		Version:     version,
		PackageType: PackageTypePypi,
		InstallPath: eggInfoPath,
	}
}
