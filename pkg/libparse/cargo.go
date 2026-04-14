package libparse

import (
	"bufio"
	"regexp"
	"strings"
)

// ParseCargoToml 解析 Cargo.toml 文件
func ParseCargoToml(content string) ([]PackageInfo, error) {
	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(content))
	inDeps := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 检查是否进入依赖块
		if strings.HasPrefix(line, "[") {
			if strings.Contains(line, "dependencies") || strings.Contains(line, "dev-dependencies") {
				inDeps = true
			} else {
				inDeps = false
			}
			continue
		}

		if !inDeps {
			continue
		}

		// 跳过空行
		if line == "" {
			continue
		}

		// 匹配依赖: package-name = "version" 或 package-name = { version = "version" }
		// 格式1: package = "version"
		re1 := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*=\s*"([^"]+)"`)
		matches1 := re1.FindStringSubmatch(line)
		if len(matches1) > 2 {
			packages = append(packages, PackageInfo{
				Ecosystem: "cargo",
				Name:      matches1[1],
				Version:   matches1[2],
			})
			continue
		}

		// 格式2: package = { version = "version" }
		re2 := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*=\s*\{.*?version\s*=\s*"([^"]+)"`)
		matches2 := re2.FindStringSubmatch(line)
		if len(matches2) > 2 {
			packages = append(packages, PackageInfo{
				Ecosystem: "cargo",
				Name:      matches2[1],
				Version:   matches2[2],
			})
			continue
		}

		// 格式3: package = { version = "version", ... }
		re3 := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*=`)
		matches3 := re3.FindStringSubmatch(line)
		if len(matches3) > 1 {
			packages = append(packages, PackageInfo{
				Ecosystem: "cargo",
				Name:      matches3[1],
			})
		}
	}

	return packages, scanner.Err()
}
