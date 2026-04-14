package pkgclient

import (
	"errors"
	"strings"
	"time"

	depsdevpb "deps.dev/api/v3"
	depsdevpbalpha "deps.dev/api/v3alpha"
	depsdev "github.com/google/osv-scalibr/depsdev"
	depsdevalpha "github.com/google/osv-scalibr/depsdev/depsdevalpha"
	"github.com/google/osv-scalibr/extractor"
	"github.com/google/osv-scalibr/purl"
)

var (
	ErrPackageNotFound   = errors.New("package not found in upstream")
	ErrPackageNotVersion = errors.New("package not found version")
	ErrRequestFailed     = errors.New("upstream request failed")
)

// RepositoryInfo 开源仓库信息（GitHub/GitLab）
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

// PackageInfo 包基础信息
type PackageInfo struct {
	Name     string
	Version  string
	PURLType string

	Description string
	HomepageURL string
	RepoURL     string // Git 地址
	Commit      string
	Licenses    []string
	ProjectID   string // deps.dev projectID
	ReleasedAt  time.Time
}

func (p *PackageInfo) ToPURL() *purl.PackageURL {
	return p.ToPackage().PURL()
}

func (p *PackageInfo) ToVersionKey() *depsdevpb.VersionKey {
	system := p.EcosystemV3()
	var version = p.Version
	if system == depsdevpb.System_GO {
		if p.Name == "stdlib" {
			version = "go" + version
		} else {
			if !strings.HasPrefix(version, "v") {
				version = "v" + version
			}
		}
	}
	return &depsdevpb.VersionKey{
		System:  system,
		Name:    p.Name,
		Version: version,
	}
}

func (p *PackageInfo) ToVersionKeyAlpha() *depsdevpbalpha.VersionKey {
	system := p.EcosystemV3Alpha()
	var version = p.Version
	if system == depsdevpbalpha.System_GO {
		if p.Name == "stdlib" {
			version = "go" + version
		} else {
			if !strings.HasPrefix(version, "v") {
				version = "v" + version
			}
		}
	}
	return &depsdevpbalpha.VersionKey{
		System:  system,
		Name:    p.Name,
		Version: version,
	}
}

func FromPackage(p *extractor.Package) *PackageInfo {
	return &PackageInfo{
		Name:     p.Name,
		Version:  p.Version,
		PURLType: p.PURLType,
		Licenses: p.Licenses,
		RepoURL:  p.SourceCode.Repo,
		Commit:   p.SourceCode.Commit,
	}
}

func FromDepsdevPackageVersion(p *depsdevpb.Package_Version) *PackageInfo {
	return &PackageInfo{
		Name:       p.VersionKey.Name,
		Version:    p.VersionKey.Version,
		PURLType:   ecosystem2purlTyp(p.VersionKey.GetSystem().String()),
		ReleasedAt: p.PublishedAt.AsTime(),
	}
}

func FromDepsdevAlphaPackageVersion(p *depsdevpbalpha.Package_Version) *PackageInfo {
	return &PackageInfo{
		Name:       p.VersionKey.Name,
		Version:    p.VersionKey.Version,
		PURLType:   ecosystem2purlTyp(p.VersionKey.GetSystem().String()),
		ReleasedAt: p.PublishedAt.AsTime(),
	}
}

func FromDepsdevAlphaVersion(p *depsdevpbalpha.Version) *PackageInfo {
	var homePage string
	var description string
	var repoUrl string
	licenses := p.GetLicenses()
	links := p.GetLinks()
	for _, link := range links {
		// HOMEPAGE\ISSUE_TRACKER\ORIGIN\SOURCE_REPO
		if link.Label == "HOMEPAGE" {
			homePage = link.Url
		}
		if link.Label == "SOURCE_REPO" {
			repoUrl = link.Url
		}
	}
	projects := p.GetRelatedProjects()
	var projectID string
	for _, project := range projects {
		if projectID == "" {
			projectID = project.ProjectKey.Id
		}
		if project.RelationType.String() == "SOURCE_REPO" {
			if repoUrl == "" {
				repoUrl = project.ProjectKey.Id
			}
		}
	}
	if strings.HasPrefix(repoUrl, "github.com") {
		repoUrl = "https://" + repoUrl
	}
	return &PackageInfo{
		Description: description,
		HomepageURL: homePage,
		RepoURL:     repoUrl,
		ProjectID:   projectID,
		Licenses:    licenses,
		Name:        p.VersionKey.Name,
		Version:     p.VersionKey.Version,
		PURLType:    ecosystem2purlTyp(p.VersionKey.GetSystem().String()),
		ReleasedAt:  p.PublishedAt.AsTime(),
	}
}

func FromDepsdevVersionKey(p *depsdevpb.VersionKey) *PackageInfo {
	return &PackageInfo{
		Name:     p.Name,
		Version:  p.Version,
		PURLType: ecosystem2purlTyp(p.GetSystem().String()),
	}
}

func FromDepsdevAlphaVersionKey(p *depsdevpbalpha.VersionKey) *PackageInfo {
	return &PackageInfo{
		Name:     p.Name,
		Version:  p.Version,
		PURLType: ecosystem2purlTyp(p.GetSystem().String()),
	}
}

func (p *PackageInfo) ToPackage() *extractor.Package {
	return &extractor.Package{
		Name:     p.Name,
		Version:  p.Version,
		PURLType: p.PURLType,
		Licenses: p.Licenses,
		SourceCode: &extractor.SourceCodeIdentifier{
			Repo:   p.RepoURL,
			Commit: p.Commit,
		},
	}
}

func (p *PackageInfo) EcosystemV3() depsdevpb.System {
	return depsdev.System[strings.ToLower(p.PURLType)]
}

func (p *PackageInfo) EcosystemV3Alpha() depsdevpbalpha.System {
	return depsdevalpha.System[strings.ToLower(p.PURLType)]
}

func ecosystem2purlTyp(s string) string {
	s = strings.ToLower(s)
	switch s {
	case "go":
		return "golang"
	case "rubygems":
		return "gem"
	case "system_upspecified":
		return ""
	default:
		return s
	}
}
