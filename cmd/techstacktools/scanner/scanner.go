package scanner

// PackageType 包类型
type PackageType string

const (
	PackageTypeGolang PackageType = "golang"
	PackageTypeNpm    PackageType = "npm"
	PackageTypePypi   PackageType = "pypi"
	PackageTypeCargo  PackageType = "cargo"
	PackageTypeAll    PackageType = "all"
)

// DependencyScope 依赖作用域
type DependencyScope string

const (
	ScopeProd  DependencyScope = "prod"  // 生产依赖
	ScopeDev   DependencyScope = "dev"   // 开发依赖
	ScopeBuild DependencyScope = "build" // 构建依赖
	ScopeTest  DependencyScope = "test"  // 测试依赖
)

// Package 代表一个本地安装的包
type Package struct {
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	PackageType  PackageType     `json:"package_type"`
	InstallPath  string          `json:"install_path,omitempty"`
	Scope        DependencyScope `json:"scope,omitempty"`         // 依赖作用域
	Registry     string          `json:"registry,omitempty"`      // 注册源
	Source       string          `json:"source,omitempty"`        // git source
	PackagePath  string          `json:"package_path,omitempty"`  // Cargo.toml 所在路径
	FeatureFlags []string        `json:"feature_flags,omitempty"` // 特性标志
}

// Scanner 包扫描器接口
type Scanner interface {
	// Scan 扫描本地安装的包
	Scan() ([]Package, error)
	// Name 返回扫描器名称
	Name() string
	// Type 返回包类型
	Type() PackageType
}
