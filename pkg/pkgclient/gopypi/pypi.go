package gopypi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type PackageUrl struct {
	Filename       string `json:"filename"`
	PackageType    string `json:"package_type"`
	PythonVersion  string `json:"python_version"`
	RequiresPython string `json:"requires_python"`
	UploadTime     string `json:"upload_time"`
	Url            string `json:"url"`
}

type PackageInfo struct {
	Name              string   `json:"name"`
	Classifiers       []string `json:"classifiers"`
	License           string   `json:"license"`
	LicenseExpression string   `json:"license_expression"`
	// Description       string     `json:"description"`
	ProjectUrls    ProjectURL `json:"project_urls"`
	Version        string     `json:"version"`
	Summary        string     `json:"summary"`
	RequiresPython string     `json:"requires_python"`
}
type ProjectURL struct {
	Changelog     string `json:"Changelog"`
	Documentation string `json:"Documentation"`
	Funding       string `json:"Funding"`
	Homepage      string `json:"Homepage"`
	Source        string `json:"Source"`
}

// Package holds all information about Python package.
type Package struct {
	Info PackageInfo `json:"info"`
	// URLS []PackageUrl `json:"urls"`
}

// Client represents PyPI client.
type Client struct {
	BaseURL    string
	HttpClient *http.Client
}

// NewClient creates a new, default PyPI client.
func NewClient() *Client {
	client := Client{
		BaseURL:    "https://pypi.org",
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
	return &client
}

// Get knows how to retrieve information from PyPI server.
// It returns info about given Python package.
func (c *Client) Get(ctx context.Context, name string) (Package, error) {
	path := fmt.Sprintf("%s/pypi/%s/json", c.BaseURL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return Package{}, err
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return Package{}, err
	}
	defer resp.Body.Close()

	var pkg Package
	if resp.StatusCode != http.StatusOK {
		return pkg, errors.New("http error status: bad request")
	}
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return Package{}, err
	}
	return pkg, nil
}

func (c *Client) GetVersion(ctx context.Context, name, version string) (Package, error) {
	path := fmt.Sprintf("%s/pypi/%s/%s/json", c.BaseURL, name, version)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return Package{}, err
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return Package{}, err
	}
	defer resp.Body.Close()

	var pkg Package
	if resp.StatusCode != http.StatusOK {
		return pkg, errors.New("http error status: bad request")
	}
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return Package{}, err
	}
	return pkg, nil
}

var DefaultClient = NewClient()

// Get takes a string representing a Python package name and returns
// detailed information about the package.
// Get internally uses default Client.
func Get(ctx context.Context, name string) (Package, error) {
	return DefaultClient.Get(ctx, name)
}
func GetVersion(ctx context.Context, name, version string) (Package, error) {
	return DefaultClient.GetVersion(ctx, name, version)
}

var usage = `Usage: pypi <Python-package-name>`

func Main() int {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		return 1
	}
	pkgName := os.Args[1]
	pkg, err := Get(context.Background(), pkgName)
	if err != nil {
		fmt.Println(usage)
		os.Exit(1)
	}
	b, err := json.Marshal(pkg)
	if err != nil {
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%+v\n", string(b))
	return 0
}
