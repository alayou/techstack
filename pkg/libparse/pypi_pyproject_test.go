package libparse

import (
	"testing"
)

const _testPyprojectFile = `[project]
name = "gitea"
version = "0.0.0"
requires-python = ">=3.10"

[dependency-groups]
dev = [
  "djlint==1.36.4",
  "yamllint==1.38.0",
]

[tool.djlint]
profile="golang"
ignore="H005,H006,H013,H016,H020,H021,H030,H031"`

func TestParsePyproject(t *testing.T) {
	pkgs, err := ParsePyproject(_testPyprojectFile)
	if err != nil {
		t.Fatal(err)
	}

	// 构建包名映射
	pkgMap := make(map[string]PackageInfo)
	for _, pkg := range pkgs {
		pkgMap[pkg.Name] = pkg
	}

	// 验证依赖组中的依赖
	expectedDevDeps := map[string]string{
		"djlint":   "==1.36.4",
		"yamllint": "==1.38.0",
	}

	for name, expectedVersion := range expectedDevDeps {
		if pkg, ok := pkgMap[name]; ok {
			if pkg.Version != expectedVersion {
				t.Errorf("Expected version %s for %s, got %s", expectedVersion, name, pkg.Version)
			}
			t.Logf("Found dev dependency: %s v%s", name, pkg.Version)
		} else {
			t.Errorf("Expected dev dependency not found: %s", name)
		}
	}

	t.Logf("Total packages found: %d", len(pkgs))
}

func TestParsePyprojectSimple(t *testing.T) {
	simplePyproject := `[project]
name = "my-package"
version = "1.0.0"
requires-python = ">=3.8"

dependencies = [
    "requests>=2.28.0",
    "flask>=2.0.0",
]

[dependency-groups]
test = [
    "pytest>=7.0.0",
    "pytest-cov>=4.0.0",
]

[tool.poetry]
name = "my-package"
version = "1.0.0"

[tool.poetry.dependencies]
python = "^3.8"
django = "^4.0.0"
`

	pkgs, err := ParsePyproject(simplePyproject)
	if err != nil {
		t.Fatal(err)
	}

	// 构建包名映射
	pkgMap := make(map[string]PackageInfo)
	for _, pkg := range pkgs {
		pkgMap[pkg.Name] = pkg
	}

	// 验证 dependencies
	if pkg, ok := pkgMap["requests"]; ok {
		if pkg.Version == "" {
			t.Error("Expected version for requests")
		}
		t.Logf("Found dependency: %s %s", pkg.Name, pkg.Version)
	} else {
		t.Error("Expected dependency requests not found")
	}

	if pkg, ok := pkgMap["flask"]; ok {
		if pkg.Version == "" {
			t.Error("Expected version for flask")
		}
		t.Logf("Found dependency: %s %s", pkg.Name, pkg.Version)
	} else {
		t.Error("Expected dependency flask not found")
	}

	// 验证 dependency-groups
	if pkg, ok := pkgMap["pytest"]; ok {
		if pkg.Version == "" {
			t.Error("Expected version for pytest")
		}
		t.Logf("Found test dependency: %s %s", pkg.Name, pkg.Version)
	} else {
		t.Error("Expected test dependency pytest not found")
	}

	// 验证 tool.poetry.dependencies
	if pkg, ok := pkgMap["django"]; ok {
		if pkg.Version == "" {
			t.Error("Expected version for django")
		}
		t.Logf("Found poetry dependency: %s %s", pkg.Name, pkg.Version)
	} else {
		t.Error("Expected poetry dependency django not found")
	}

	t.Logf("Packages found: %v", pkgs)
}
