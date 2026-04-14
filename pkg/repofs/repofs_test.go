package repofs_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	goversion "github.com/hashicorp/go-version"

	"github.com/alayou/techstack/pkg/repofs"
	"github.com/spf13/afero"
)

func TestScanFromFs(t *testing.T) {
	rootFs := afero.NewReadOnlyFs(afero.NewBasePathFs(afero.NewOsFs(), "../../"))
	res := repofs.ScanFromFs(t.Context(), afero.NewIOFS(rootFs))
	if res.Status.Status != 1 {
		t.Error(res.Status.FailureReason)
	}
	for _, s := range res.PluginStatus {
		if s.Status.Status != 1 {
			t.Error(s.Status.FailureReason)
		}
		t.Logf("Name=%s Version=%d Status=%s", s.Name, s.Version, s.Status.String())
	}
	for _, p := range res.Inventory.Packages {
		t.Logf("Name=%s Version=%s PURLTYPe=%s PURL=%s  FromFile %s", p.Name, p.Version, p.PURLType, p.PURL().String(), p.Locations[0])
	}
	t.Logf("耗时:%s", res.EndTime.Sub(res.StartTime).String())
}

func TestGitCloneFromRemoteToFs(t *testing.T) {
	rootFs, _, err := repofs.GitCloneFromRemoteToFs(t.Context(), "https://github.com/go-git/go-git.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	ls, err := afero.ReadDir(rootFs, "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) == 0 {
		t.Fatal("Not Found any Files")
	}
	for _, file := range ls {
		t.Logf("FileName=%s FileSize=%d", file.Name(), file.Size())
	}
}

func TestScanGitRepo(t *testing.T) {
	now := time.Now()
	defer func() {
		end := time.Now()
		t.Logf("总耗时:%s", end.Sub(now).String())
	}()
	res, err := repofs.ScanGitRepo(t.Context(), "https://github.com/go-git/go-git.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	if res.Status.Status != 1 {
		t.Error(res.Status.FailureReason)
	}
	for _, s := range res.PluginStatus {
		if s.Status.Status != 1 {
			t.Error(s.Status.FailureReason)
		}
		t.Logf("Name=%s Version=%d Status=%s", s.Name, s.Version, s.Status.String())
	}
	for _, p := range res.Inventory.Packages {
		t.Logf("Name=%s Version=%s PURLTYPe=%s PURL=%s  FromFile %s", p.Name, p.Version, p.PURLType, p.PURL().String(), p.Locations[0])
	}
	t.Logf("分析耗时:%s", res.EndTime.Sub(res.StartTime).String())
}

func TestVersion(t *testing.T) {
	v, err := goversion.NewVersion("1")
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "1.0.0" {
		t.Fatal("parse Error")
	}
	v, err = goversion.NewVersion("v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "1.2.3" {
		t.Fatal("parse Error")
	}
}

func TestGitCloneFromRemoteToOsFs(t *testing.T) {
	dir, _ := os.Getwd()
	dir = filepath.Join(dir, ".cache")
	defer os.RemoveAll(dir)
	os.MkdirAll(dir, os.ModePerm)
	err := repofs.GitCloneFromRemoteToOsFs(t.Context(), "https://bgithub.xyz/abhigyanpatwari/GitNexus.git", "main", dir)
	if err != nil {
		t.Fatal(err)
	}
}
