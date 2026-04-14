package gogithub

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	scalibrversion "github.com/google/osv-scalibr/version"
)

var (
	ErrPackageNotFound   = errors.New("package not found in upstream")
	ErrPackageNotVersion = errors.New("package not found version")
	ErrRepoNotFound      = errors.New("repository not found")
	ErrInvalidRepoURL    = errors.New("invalid repository url")
	ErrRequestFailed     = errors.New("upstream request failed")
	ErrorRepoURLInvalid  = errors.New("ErrorRepoURLInvalid")
)

type RepositoryInfo struct {
	Platform      string // github / gitlab
	Owner         string
	Name          string
	FullName      string // owner/name
	Description   string
	HomepageURL   string
	Stars         int
	Forks         int
	License       string
	DefaultBranch string
	URL           string // 网页地址
	GitURL        string // clone 地址
	Language      string // clone 地址
}

// GetGithubRepoInfo identity支持格式 onwer/repo , github.com/onwer/rep , https://github.com/onwer/rep , https://github.com/onwer/rep.git
func GetGithubRepoInfo(ctx context.Context, identity string) (repo *RepositoryInfo, err error) {
	identity = strings.TrimSuffix(identity, ".git")
	identity = strings.TrimPrefix(identity, "https://github.com/")
	identity = strings.TrimPrefix(identity, "github.com/")
	identity = strings.Trim(identity, "/")
	fulleName := strings.Split(identity, "/")
	if len(fulleName) != 2 {
		err = ErrorRepoURLInvalid
		return
	}
	owner, name := fulleName[0], fulleName[1]

	var res struct {
		Description   string                `json:"description"`
		Homepage      string                `json:"homepage"`
		Stargazers    int                   `json:"stargazers_count"`
		Forks         int                   `json:"forks_count"`
		License       struct{ Name string } `json:"license"`
		DefaultBranch string                `json:"default_branch"`
		HtmlURL       string                `json:"html_url"`
		GitURL        string                `json:"git_url"`
		Language      string                `json:"language"`
	}

	url := "https://api.github.com/repos/" + owner + "/" + name
	headers := map[string]string{"User-Agent": "osv-scalibr/" + scalibrversion.ScannerVersion}
	client := http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return
	}

	for k, val := range headers {
		req.Header.Set(k, val)
	}

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == 404 {
			err = ErrRepoNotFound
			return
		}
		err = ErrRequestFailed
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	return &RepositoryInfo{
		Platform:      "github",
		Owner:         owner,
		Name:          name,
		FullName:      owner + "/" + name,
		Description:   res.Description,
		HomepageURL:   res.Homepage,
		Stars:         res.Stargazers,
		Forks:         res.Forks,
		License:       res.License.Name,
		DefaultBranch: res.DefaultBranch,
		URL:           res.HtmlURL,
		GitURL:        res.GitURL,
		Language:      res.Language,
	}, nil

}
