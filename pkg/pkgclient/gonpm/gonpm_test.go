package gonpm_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/alayou/techstack/pkg/pkgclient/gonpm"
)

func readFile(filepath string) ([]byte, error) {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func testServer(t *testing.T) *httptest.Server {
	t.Helper()

	content, err := readFile("testdata/response_claude-code.json")
	if err != nil {
		t.Fatal(err)
	}

	f := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, string(content))
	}
	return httptest.NewServer(http.HandlerFunc(f))
}

func TestPackageService_Get(t *testing.T) {
	t.Parallel()

	server := testServer(t)
	defer server.Close()

	client := gonpm.NewClient()
	client.BaseURL = server.URL

	pythonPackage := "claude-code"
	got, err := client.Get(context.Background(), pythonPackage)
	if err != nil {
		t.Errorf("client.Package.Get(%q) = %v", pythonPackage, got)
	}

	want := "claude-code"
	if got.Name != want {
		t.Errorf("client.Package.Get(%q) = %s; want %s", pythonPackage, got.Name, want)
	}

	want = "MIT"
	if got.License != want {
		t.Errorf("client.Package.Get(%q) = %s; want %s", pythonPackage, got.License, want)
	}

	want = "https://registry.npmjs.org/claude-code/-/claude-code-1.0.0.tgz"
	if got.Dist.Tarball != want {
		t.Errorf("client.Package.Get(%q) = %s; want %s", pythonPackage, got.Dist.Tarball, want)
	}

	want = "git+https://github.com/anthropics/claude-code.git"
	if got.Repository.Url != want {
		t.Errorf("client.Package.Get(%q) = %s; want %s", pythonPackage, got.Repository.Url, want)
	}
}
