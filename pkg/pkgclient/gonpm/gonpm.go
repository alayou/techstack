package gonpm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	HttpClient *http.Client
}

func NewClient() *Client {
	client := Client{
		BaseURL:    "https://registry.npmjs.org",
		HttpClient: &http.Client{Timeout: 10 * time.Second},
	}
	return &client
}

func (c *Client) Get(ctx context.Context, name string) (Package, error) {
	path := fmt.Sprintf("%s/%s/latest", c.BaseURL, name)
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
	path := fmt.Sprintf("%s/%s/%s", c.BaseURL, name, version)
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

func (c *Client) GetVersions(ctx context.Context, name string) (map[string]Version, error) {
	path := fmt.Sprintf("%s/%s", c.BaseURL, name)
	req, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return map[string]Version{}, err
	}
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return map[string]Version{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("http error status: bad request")
	}
	var pkg Package
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return map[string]Version{}, err
	}
	return pkg.Versions, nil
}

var DefaultClient = NewClient()

func Get(ctx context.Context, name string) (Package, error) {
	return DefaultClient.Get(ctx, name)
}
func GetVersion(ctx context.Context, name, version string) (Package, error) {
	return DefaultClient.GetVersion(ctx, name, version)
}
func GetVersions(ctx context.Context, name string) (map[string]Version, error) {
	return DefaultClient.GetVersions(ctx, name)
}
