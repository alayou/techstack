package libparse

import (
	"bufio"
	"regexp"
	"strings"
)

// ParsePyproject 解析 pyproject.toml 文件
func ParsePyproject(content string) ([]PackageInfo, error) {
	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(content))

	var inProjectDeps, inDevDeps, inDepsArray bool

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// 检查是否是节头
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			section := strings.Trim(trimmedLine, "[]")

			// [project.dependencies] - 项目依赖
			if section == "project.dependencies" || section == "project.optional-dependencies" {
				inProjectDeps = true
				inDevDeps = false
				inDepsArray = false
				continue
			}

			// [project.optional-dependencies] - 可选依赖
			if strings.HasPrefix(section, "project.optional-dependencies") {
				inProjectDeps = true
				inDevDeps = false
				inDepsArray = false
				continue
			}

			// [dependency-groups] - 依赖组
			if section == "dependency-groups" {
				inProjectDeps = false
				inDevDeps = true
				inDepsArray = false
				continue
			}

			// [tool.poetry.dependencies] - Poetry 依赖
			if strings.HasPrefix(section, "tool.poetry.dependencies") {
				inProjectDeps = true
				inDevDeps = false
				inDepsArray = false
				continue
			}

			// [tool.pdm.dev.dependencies] - PDM 开发依赖
			if strings.HasPrefix(section, "tool.pdm.dev.dependencies") {
				inProjectDeps = false
				inDevDeps = true
				inDepsArray = false
				continue
			}

			// [tool.pdm.dependencies] - PDM 依赖
			if strings.HasPrefix(section, "tool.pdm.dependencies") {
				inProjectDeps = true
				inDevDeps = false
				inDepsArray = false
				continue
			}

			// [project] - 项目根节，检查 dependencies 字段
			if section == "project" {
				inProjectDeps = true // 项目根节内识别dependencies数组
				inDevDeps = false
				inDepsArray = false
				continue
			}

			// 其他工具配置 - 重置状态
			if strings.HasPrefix(section, "tool.") {
				inProjectDeps = false
				inDevDeps = false
				inDepsArray = false
				continue
			}

			// 重置状态
			inProjectDeps = false
			inDevDeps = false
			inDepsArray = false
			continue
		}

		// 检查是否是依赖数组开始
		if inProjectDeps || inDevDeps {
			// 检查是否是数组开始 (xxx = [)
			// 例如: dependencies = [ 或 test = [
			if strings.Contains(trimmedLine, "=") && strings.Contains(trimmedLine, "[") {
				// 提取等号左边的键名
				parts := strings.Split(trimmedLine, "=")
				if len(parts) >= 2 {
					key := strings.TrimSpace(parts[0])
					// 只有 dependencies 才是依赖数组，其他是子节
					if key == "dependencies" {
						inDepsArray = true
						continue
					}
					// 其他 xxx = [ 格式跳过
					continue
				}
			}
		}

		// 检查数组是否结束
		if inDepsArray {
			if strings.Contains(trimmedLine, "]") {
				inDepsArray = false
			}
			// 跳过空行
			if trimmedLine == "" {
				continue
			}
			// 处理数组中的依赖项（缩进行）
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				pkg := extractPythonPackage(trimmedLine)
				if pkg.Name != "" {
					packages = append(packages, pkg)
				}
			}
			continue
		}

		// 只有在依赖块内才���析
		if !inProjectDeps && !inDevDeps {
			continue
		}

		// 跳过空行和注释
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// 跳过 continuation (缩进行) - 非数组格式
		if strings.HasPrefix(trimmedLine, " ") || strings.HasPrefix(trimmedLine, "\t") {
			continue
		}

		// 解析单行依赖
		// 格式: package-name = "version" 或 package-name = { version = "version" }
		// 或: "package" (在列表中)
		pkg := extractPythonPackage(trimmedLine)
		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}

	return packages, scanner.Err()
}

// extractPythonPackage 从行中提取 Python 包信息
func extractPythonPackage(line string) PackageInfo {
	line = strings.TrimSpace(line)

	// 跳过明显不是依赖的行
	lowerLine := strings.ToLower(line)
	if lowerLine == "name" || lowerLine == "version" || lowerLine == "description" ||
		lowerLine == "authors" || lowerLine == "readme" || lowerLine == "license" ||
		lowerLine == "requires-python" || lowerLine == "keywords" ||
		lowerLine == "homepage" || lowerLine == "repository" || lowerLine == "documentation" {
		return PackageInfo{}
	}

	// 去掉首尾引号和逗号
	for len(line) > 0 && (strings.HasPrefix(line, "\"") || strings.HasPrefix(line, "'") || strings.HasPrefix(line, ",")) {
		line = line[1:]
	}
	for len(line) > 0 && (strings.HasSuffix(line, "\"") || strings.HasSuffix(line, "'") || strings.HasSuffix(line, ",")) {
		line = line[:len(line)-1]
	}

	// 跳过非依赖行
	if strings.HasPrefix(line, "#") {
		return PackageInfo{}
	}

	// 格式1: package==version 或 package>=version (不带引号)
	re1 := regexp.MustCompile(`^([a-zA-Z0-9_-]+)(==|>=|<=|!=|~=|>|<)(.+)$`)
	matches1 := re1.FindStringSubmatch(line)
	if len(matches1) > 1 {
		return PackageInfo{
			Ecosystem: "pypi",
			Name:      matches1[1],
			Version:   matches1[2] + matches1[3],
		}
	}

	// 格式2: package = "version" (TOML格式)
	re2 := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*=\s*"?([^"'\s,]+)"?`)
	matches2 := re2.FindStringSubmatch(line)
	if len(matches2) > 2 {
		// 过滤掉项目元数据字段
		switch matches2[1] {
		case "name", "version", "description", "authors", "readme", "license",
			"requires-python", "keywords", "homepage", "repository", "documentation",
			"classifiers", "urls", "dependencies", "optional-dependencies":
			return PackageInfo{}
		}
		return PackageInfo{
			Ecosystem: "pypi",
			Name:      matches2[1],
			Version:   matches2[2],
		}
	}

	// 格式3: package = { version = "version", ... }
	re3 := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*=\s*\{.*?version\s*=\s*"?([^"'\s,]+)"?`)
	matches3 := re3.FindStringSubmatch(line)
	if len(matches3) > 2 {
		return PackageInfo{
			Ecosystem: "pypi",
			Name:      matches3[1],
			Version:   matches3[2],
		}
	}

	// 格式4: 裸包名 (如 requests, flask)
	re4 := regexp.MustCompile(`^([a-zA-Z0-9_-]+)$`)
	matches4 := re4.FindStringSubmatch(line)
	if len(matches4) > 1 {
		return PackageInfo{
			Ecosystem: "pypi",
			Name:      matches4[1],
			Version:   "",
		}
	}

	return PackageInfo{}
}
