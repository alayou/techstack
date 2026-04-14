package libparse

import (
	"bufio"
	"errors"
	"os"
	"regexp"
	"strings"

	"golang.org/x/mod/modfile"
)

// ParseGoMod 使用 golang.org/x/mod/modfile 解析 go.mod 文件
func ParseGoMod(content string) ([]PackageInfo, error) {
	// modfile.ParseLax 可以在没有严格语法要求的情况下解析
	lines := strings.Split(content, "\n")
	var filteredLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过 module 和 go 版本声明
		if strings.HasPrefix(trimmed, "module ") || strings.HasPrefix(trimmed, "go ") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	// 在前面加上 module 声明
	modifiedContent := "module placeholder\n" + strings.Join(filteredLines, "\n")

	f, err := modfile.ParseLax("go.mod", []byte(modifiedContent), nil)
	if err != nil {
		// 如果解析失败，回退到正则解析
		return parseGoModFallback(content)
	}

	var packages []PackageInfo

	// 添加 require 块中的依赖
	for _, req := range f.Require {
		packages = append(packages, PackageInfo{
			Ecosystem: "go",
			Name:      req.Mod.Path,
			Version:   req.Mod.Version,
		})
	}

	// 添加 replace 指令中的原包
	for _, rep := range f.Replace {
		packages = append(packages, PackageInfo{
			Ecosystem: "go",
			Name:      rep.Old.Path,
			Version:   rep.Old.Version,
		})
	}

	// ParseLax 可能不会解析 replace，使用正则提取
	reReplace := regexp.MustCompile(`replace\s+(\S+)\s+=>\s+\S+\s+(v?[\d.]+)`)
	for _, line := range strings.Split(content, "\n") {
		matches := reReplace.FindStringSubmatch(line)
		if len(matches) > 2 {
			packages = append(packages, PackageInfo{
				Ecosystem: "go",
				Name:      matches[1],
				Version:   matches[2],
			})
		}
	}

	return packages, nil
}

// parseGoModFallback 回退到正则解析
func parseGoModFallback(content string) ([]PackageInfo, error) {
	var packages []PackageInfo

	scanner := bufio.NewScanner(strings.NewReader(content))
	var inRequireBlock bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行
		if line == "" {
			continue
		}

		// 处理注释
		if strings.HasPrefix(line, "//") {
			continue
		}

		// 处理 require 块开始
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}

		// 处理 require 块结束
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		// 处理 replace 指令
		if strings.HasPrefix(line, "replace ") {
			re := regexp.MustCompile(`replace\s+(\S+)\s+=>.*?v?([\d.]+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				version := ""
				if len(matches) > 2 {
					version = matches[2]
				}
				packages = append(packages, PackageInfo{
					Ecosystem: "go",
					Name:      matches[1],
					Version:   version,
				})
			}
			continue
		}

		// 跳过其他指令
		if strings.HasPrefix(line, "module ") ||
			strings.HasPrefix(line, "go ") ||
			strings.HasPrefix(line, "tool ") ||
			strings.HasPrefix(line, "exclude (") ||
			strings.HasPrefix(line, "ignore (") ||
			strings.HasPrefix(line, "retract (") ||
			strings.HasPrefix(line, "replace (") {
			continue
		}

		// 解析单行依赖
		// 格式: package-name v1.2.3 或 package-name v1.2.3 // indirect
		re := regexp.MustCompile(`^(\S+)\s+v?([\d.]+)`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := ""
			if len(matches) > 2 {
				version = matches[2]
			}
			packages = append(packages, PackageInfo{
				Ecosystem: "go",
				Name:      matches[1],
				Version:   version,
			})
		}
	}

	return packages, scanner.Err()
}

// ParseGoModFile 解析 go.mod 文件（从文件路径）
func ParseGoModFile(filePath string) ([]PackageInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.New("读取文件失败: " + err.Error())
	}
	return ParseGoMod(string(data))
}
