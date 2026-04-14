package gogitlab

import (
	"context"
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
}

func GetGithubRepoInfo(ctx context.Context, identity string) (repo *RepositoryInfo, err error) {
	identity = strings.TrimSuffix(identity, ".git")
	identity = strings.TrimPrefix(identity, "https://gitlab.com/")
	identity = strings.TrimPrefix(identity, "gitlab.com/")
	identity = strings.Trim(identity, "/")
	fulleName := strings.Split(identity, "/")
	if len(fulleName) != 2 {
		err = ErrorRepoURLInvalid
		return
	}
	owner, name := fulleName[0], fulleName[1]
	full := owner + "%2F" + name

	var res struct {
		Description   string `json:"description"`
		Homepage      string `json:"web_url"`
		Stars         int    `json:"star_count"`
		Forks         int    `json:"forks_count"`
		License       string `json:"license"`
		DefaultBranch string `json:"default_branch"`
		WebURL        string `json:"web_url"`
		GitURL        string `json:"http_url_to_repo"`
	}

	url := "https://gitlab.com/api/v4/projects/" + full
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

	return &RepositoryInfo{
		Platform:      "gitlab",
		Owner:         owner,
		Name:          name,
		FullName:      owner + "/" + name,
		Description:   res.Description,
		HomepageURL:   res.Homepage,
		Stars:         res.Stars,
		Forks:         res.Forks,
		License:       res.License,
		DefaultBranch: res.DefaultBranch,
		URL:           res.WebURL,
		GitURL:        res.GitURL,
	}, nil
}
