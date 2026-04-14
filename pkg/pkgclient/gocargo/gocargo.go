package gocargo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/go-getter"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		baseURL: "https://crates.io",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Search 搜索包
func (c *Client) Search(ctx context.Context, keyword string) ([]Crate, error) {
	path := fmt.Sprintf("%s/api/v1/crates?q=%s", c.baseURL, keyword)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Crates []Crate `json:"crates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Crates, nil
}

// GetPackage 获取单个包详情
func (c *Client) GetPackage(ctx context.Context, crateName string) (*Crate, error) {
	path := fmt.Sprintf("%s/api/v1/crates/%s?include=default_version", c.baseURL, crateName)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("http error status: bad request")
	}

	var res struct {
		Crate    Crate     `json:"crate"`
		Versions []Version `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	res.Crate.Versions = res.Versions
	return &res.Crate, nil
}

// GetVersion
func (c *Client) GetVersion(ctx context.Context, crateName, version string) (*Version, error) {
	//
	path := fmt.Sprintf("%s/api/v1/crates/%s/%s", c.baseURL, crateName, version)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("http error status: bad request")
	}
	var res struct {
		Version Version `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res.Version, nil
}

// Versions 获取包的所有版本
func (c *Client) Versions(ctx context.Context, name string, size int) ([]Version, error) {
	addr, err := url.Parse(fmt.Sprintf("%s/api/v1/crates/%s/versions", c.baseURL, name))
	if err != nil {
		return nil, err
	}
	values := addr.Query()
	if size == 0 {
		size = 5
	}
	values.Set("per_page", strconv.Itoa(size))
	values.Set("sort", "semver")
	addr.RawQuery = values.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", addr.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Versions []Version `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Versions, nil
}

func (c *Client) GetDependencies(ctx context.Context, name string, version string) ([]CrateDependency, error) {
	addr, err := url.Parse(fmt.Sprintf("%s/api/v1/crates/%s/%s/dependencies", c.baseURL, name, version))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", addr.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Dependencies []CrateDependency `json:"dependencies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Dependencies, nil
}

func (c *Client) GetSourceGZipBytes(ctx context.Context, name, version string, dst io.Writer) error {
	path := fmt.Sprintf("%s/crates/%s/%s/download", c.baseURL, name, version)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(dst, resp.Body)
	return err
}

func (c *Client) GetSourceZipFile(ctx context.Context, name, version, dst string) error {
	path := fmt.Sprintf("%s/crates/%s/%s/download", c.baseURL, name, version)
	return getter.GetFile(dst, path, getter.WithContext(ctx))
}

var DefaultClient = New()

func Search(ctx context.Context, keyword string) ([]Crate, error) {
	return DefaultClient.Search(ctx, keyword)
}

func GetPackage(ctx context.Context, crateName string) (*Crate, error) {
	return DefaultClient.GetPackage(ctx, crateName)
}

func GetVersion(ctx context.Context, name, version string) (*Version, error) {
	return DefaultClient.GetVersion(ctx, name, version)

}
func Versions(ctx context.Context, name string, size int) ([]Version, error) {
	return DefaultClient.Versions(ctx, name, size)
}

func GetDependencies(ctx context.Context, name string, version string) ([]CrateDependency, error) {
	return DefaultClient.GetDependencies(ctx, name, version)
}
