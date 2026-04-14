package gogo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-getter"
)

type Client struct {
	BaseURL    string
	HttpClient *http.Client
}
type options struct {
	baseUrl string
}
type Option func(*options)

func WithProxy(baseUrl string) Option {
	return func(o *options) {
		o.baseUrl = baseUrl
	}
}
func NewClient(opts ...Option) *Client {
	var options = &options{
		baseUrl: "https://proxy.golang.org",
	}
	for _, opt := range opts {
		opt(options)
	}
	client := Client{
		BaseURL:    options.baseUrl,
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
	return &client
}

type VersionInfo struct {
	Version string `json:"Version"`
	Time    string `json:"Time"`
}

func (c *Client) GetPackage(ctx context.Context, name string) (VersionInfo, error) {
	path := fmt.Sprintf("%s/%s/@latest", c.BaseURL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return VersionInfo{}, err
	}
	if err != nil {
		return VersionInfo{}, err
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return VersionInfo{}, err
	}
	defer resp.Body.Close()
	var out VersionInfo
	if resp.StatusCode != http.StatusOK {
		return out, errors.New("http error status: bad request")
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return VersionInfo{}, err
	}
	return out, nil
}
func (c *Client) GetVersions(ctx context.Context, name string) ([]string, error) {
	path := fmt.Sprintf("%s/%s/@v/list", c.BaseURL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return []string{}, err
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []string{}, errors.New("http error status: bad request")
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}
	if len(raw) == 0 {
		return []string{}, nil
	}
	return strings.Split(string(raw), "\n"), nil
}

func (c *Client) GetVersion(ctx context.Context, name, version string) (VersionInfo, error) {
	path := fmt.Sprintf("%s/%s/@v/%s.info", c.BaseURL, name, version)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return VersionInfo{}, err
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return VersionInfo{}, err
	}
	defer resp.Body.Close()

	var out VersionInfo
	if resp.StatusCode != http.StatusOK {
		return out, errors.New("http error status: bad request")
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return VersionInfo{}, err
	}
	return out, nil
}

func (c *Client) GetModFile(ctx context.Context, name, version string) (string, error) {
	path := fmt.Sprintf("%s/%s/@v/%s.mod", c.BaseURL, name, version)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}
	res, err := c.HttpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// GetSourceZipFile
func (c *Client) GetSourceZipFile(ctx context.Context, name, version, dst string) error {
	path := fmt.Sprintf("%s/%s/@v/%s.zip", c.BaseURL, name, version)
	return getter.GetFile(dst, path, getter.WithContext(ctx))
}

func (c *Client) GetSourceZipBytes(ctx context.Context, name, version string, dst io.Writer) error {
	path := fmt.Sprintf("%s/%s/@v/%s.zip", c.BaseURL, name, version)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	res, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.Copy(dst, res.Body)
	return err
}

var DefaultClient = NewClient()

func GetPackage(ctx context.Context, name string) (VersionInfo, error) {
	return DefaultClient.GetPackage(ctx, name)
}
func GetVersions(ctx context.Context, name string) ([]string, error) {
	return DefaultClient.GetVersions(ctx, name)
}

func GetVersion(ctx context.Context, name, version string) (VersionInfo, error) {
	return DefaultClient.GetVersion(ctx, name, version)
}

func GetModFile(ctx context.Context, name, version string) (string, error) {
	return DefaultClient.GetModFile(ctx, name, version)
}

func GetSourceZipFile(ctx context.Context, name, version, dst string) error {
	return DefaultClient.GetSourceZipFile(ctx, name, version, dst)
}

func GetSourceZipBytes(ctx context.Context, name, version string, dst io.Writer) error {
	return DefaultClient.GetSourceZipBytes(ctx, name, version, dst)
}
