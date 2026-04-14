package libparse

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScannerScanPath(t *testing.T) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "scanner-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 go.mod 文件
	goModContent := `module github.com/test/project

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.4.0
)
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建 package.json 文件
	pkgJsonContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "lodash": "4.17.21",
    "axios": "^1.6.0"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}
`
	pkgJsonPath := filepath.Join(tmpDir, "package.json")
	if err := os.WriteFile(pkgJsonPath, []byte(pkgJsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 创建 pyproject.toml 文件
	pyprojectContent := `[project]
name = "test-project"
version = "1.0.0"

dependencies = [
    "requests>=2.28.0",
    "flask>=2.0.0",
]
`
	pyprojectPath := filepath.Join(tmpDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(pyprojectContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 执行扫描
	scanner := NewScanner()
	result, err := scanner.ScanPath(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// 验证结果
	t.Logf("Total packages: %d", result.Summary.TotalPackages)
	t.Logf("By ecosystem: %v", result.Summary.ByEcosystem)
	t.Logf("Packages: %v", result.Packages)

	if result.Summary.TotalPackages == 0 {
		t.Error("expected packages to be found")
	}
}

func TestScannerScanInput(t *testing.T) {
	// 测试本地路径扫描
	tmpDir, err := os.MkdirTemp("", "scanner-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建 Cargo.toml
	cargoContent := `[package]
name = "test-crate"
version = "0.1.0"

[dependencies]
serde = "1.0"

[dev-dependencies]
criterion = "0.5"
`
	cargoPath := filepath.Join(tmpDir, "Cargo.toml")
	if err := os.WriteFile(cargoPath, []byte(cargoContent), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := NewScanner()
	result, err := scanner.ScanInput(context.Background(), tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// 验证 Cargo 包
	var cargoPkgs int
	for _, pkg := range result.Packages {
		if pkg.Ecosystem == "cargo" {
			cargoPkgs++
		}
	}

	if cargoPkgs == 0 {
		t.Error("expected cargo packages to be found")
	}
}
