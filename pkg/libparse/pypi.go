package libparse

import (
	"bufio"
	"regexp"
	"strings"
)

// ParseRequirementsTxt 解析 requirements.txt 文件
func ParseRequirementsTxt(content string) ([]PackageInfo, error) {
	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 跳过 options 如 -r, -f, -e 等
		if strings.HasPrefix(line, "-") {
			continue
		}

		// 匹配包名和版本
		// 格式: package-name==1.2.3 或 package-name>=1.0.0
		re := regexp.MustCompile(`^([a-zA-Z0-9_-]+)(?:==|>=|<=|!=|~=)?(.*)$`)
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := ""
			if len(matches) > 2 {
				version = matches[2]
			}
			packages = append(packages, PackageInfo{
				Ecosystem: "pypi",
				Name:      matches[1],
				Version:   version,
			})
		}
	}

	return packages, scanner.Err()
}
