package libparse

// PackageInfo 包信息
type PackageInfo struct {
	// Ecosystem 包管理器类型
	Ecosystem string
	// Name 包名称
	Name string
	// Version 包版本
	Version string
}

// ParseDepsFile 解析依赖文件，返回包列表
func ParseDepsFile(content, fileType string) ([]PackageInfo, error) {
	switch fileType {
	case "go.mod":
		return ParseGoMod(content)
	case "package.json":
		return ParsePackageJson(content)
	case "pyproject.toml":
		return ParsePyproject(content)
	case "requirements.txt":
		return ParseRequirementsTxt(content)
	case "Cargo.toml":
		return ParseCargoToml(content)
	default:
		return nil, nil
	}
}
