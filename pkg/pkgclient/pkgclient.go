package pkgclient

import (
	"context"
	"strings"

	"github.com/alayou/techstack/pkg/chain"
	"github.com/alayou/techstack/pkg/pkgclient/depsdev"
	"github.com/alayou/techstack/pkg/pkgclient/gocargo"
	"github.com/alayou/techstack/pkg/pkgclient/gogo"
	"github.com/alayou/techstack/pkg/pkgclient/gonpm"
	"github.com/alayou/techstack/pkg/pkgclient/gopypi"
	"github.com/google/osv-scalibr/purl"
)

type PkgClient struct {
}

func NewPkgClient() *PkgClient {
	return &PkgClient{}
}

// GetPackageInfo 获取包信息，system和name 必填，version 为空则查询最新版本的包
func (s *PkgClient) GetPackageInfo(ctx context.Context, system, name, version string) (*PackageInfo, error) {
	switch system {
	case "npm":
		return s.GetNpmPackageInfoByRegistry(ctx, name, version)
	case "cargo":
		return s.GetCargoPackageInfoByRegistry(ctx, name, version)
	case "pypi":
		return s.GetPypiPackageInfoByRegistry(ctx, name, version)
	case "golang", "go":
		return s.GetGolangPackageInfoByRegistry(ctx, name, version)
	default:
		if version == "" {
			return s.GetPackageInfoByDepsdev(ctx, system, name)
		}
		return s.GetPackageInfoByDepsdevVersion(ctx, system, name, version)
	}
}
func (s *PkgClient) GetNpmPackageInfoByRegistry(ctx context.Context, name, version string) (*PackageInfo, error) {
	var pkg gonpm.Package
	var err error
	if version != "" {
		pkg, err = gonpm.GetVersion(ctx, name, version)
	} else {
		pkg, err = gonpm.Get(ctx, name)
	}
	if err != nil {
		return nil, err
	}
	return &PackageInfo{
		Name:        pkg.Name,
		Description: pkg.Description,
		HomepageURL: pkg.Homepage,
		RepoURL:     pkg.Repository.Url,
		Licenses:    []string{pkg.License},
		Version:     pkg.DistTags.Latest,
	}, nil
}

func (s *PkgClient) GetGolangPackageInfoByRegistry(ctx context.Context, name, version string) (*PackageInfo, error) {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	client := gogo.NewClient(gogo.WithProxy("https://goproxy.cn"))
	if version != "" {
		return s.GetPackageInfoByDepsdevVersion(ctx, "go", name, version)
	}
	pkg, err := client.GetPackage(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.GetPackageInfoByDepsdevVersion(ctx, "go", name, pkg.Version)
}

func (s *PkgClient) GetCargoPackageInfoByRegistry(ctx context.Context, name, version string) (*PackageInfo, error) {
	if version != "" {
		v, err := gocargo.GetVersion(ctx, name, version)
		if err != nil {
			return nil, err
		}
		return &PackageInfo{
			Name:        v.Crate,
			Description: v.Description,
			HomepageURL: v.GetHomepageURL(),
			RepoURL:     v.Repository,
			Licenses:    []string{v.License},
			Version:     version,
		}, nil
	}
	pkg, err := gocargo.GetPackage(ctx, name)
	if err != nil {
		return nil, err
	}
	return &PackageInfo{
		Name:        pkg.Name,
		Description: pkg.Description,
		HomepageURL: pkg.GetHomepageURL(),
		RepoURL:     pkg.Repository,
		Licenses:    []string{pkg.GetLicence()},
		Version:     pkg.DefaultVersion,
	}, nil
}

func (s *PkgClient) GetPypiPackageInfoByRegistry(ctx context.Context, name, version string) (*PackageInfo, error) {
	var pkg gopypi.Package
	var err error
	if version != "" {
		pkg, err = gopypi.GetVersion(ctx, name, version)
	} else {
		pkg, err = gopypi.Get(ctx, name)
	}
	if err != nil {
		return nil, err
	}
	if pkg.Info.License == "" {
		pkg.Info.License = pkg.Info.LicenseExpression
	}

	return &PackageInfo{
		Name:        pkg.Info.Name,
		Description: pkg.Info.Summary,
		HomepageURL: pkg.Info.ProjectUrls.Homepage,
		RepoURL:     pkg.Info.ProjectUrls.Source,
		Licenses:    []string{pkg.Info.License},
		Version:     pkg.Info.Version,
	}, nil
}

func (s *PkgClient) GetPackageInfoByDepsdev(ctx context.Context, system, name string) (*PackageInfo, error) {
	var depsClient *depsdev.DepsDevClient
	depsClient, err := depsdev.NewClient()
	if err != nil {
		return nil, err
	}
	pkg, err := depsClient.GetPackage(ctx, system, name)
	if err != nil {
		return nil, err
	}
	if len(pkg.Versions) == 0 {
		return nil, ErrPackageNotVersion
	}
	ver := pkg.Versions[len(pkg.GetVersions())-1]
	var latestVersion = ver.VersionKey.Version
	return s.GetPackageInfoByDepsdevVersion(ctx, system, name, latestVersion)
}

func (s *PkgClient) GetPackageInfoListByDepsdev(ctx context.Context, vbr []depsdev.VersionBatchReq) ([]*PackageInfo, error) {
	var depsClient *depsdev.DepsDevClient
	depsClient, err := depsdev.NewClient()
	if err != nil {
		return nil, err
	}
	vbr = chain.From(vbr).UniqueBy(func(p depsdev.VersionBatchReq) any {
		return struct {
			Name     string
			PurlType string
		}{p.Name, p.System}
	}).ToSlice()
	versions, err := depsClient.GetVersionBatch(ctx, "", vbr)
	if err != nil {
		return nil, err
	}
	var projectIDs []string
	var pkgs []*PackageInfo
	for i := range versions {
		if versions[i].Version == nil {
			continue
		}
		pkg := FromDepsdevAlphaVersion(versions[i].Version)
		pkgs = append(pkgs, pkg)
		if pkg.ProjectID != "" {
			projectIDs = append(projectIDs, pkg.ProjectID)
		}
	}
	if len(projectIDs) == 0 {
		return pkgs, err
	}
	res, err := depsClient.GetProjectsBatch(ctx, "", projectIDs)
	if err != nil {
		return nil, err
	}
	for _, p := range res {
		project := p.Project
		if p.Project == nil {
			continue
		}
		homePage := project.Homepage
		description := project.Description
		for i, pkg := range pkgs {
			if pkg.ProjectID == p.Project.ProjectKey.Id {
				pkg.Description = description
				pkg.HomepageURL = homePage
				pkgs[i] = pkg
			}
		}
	}
	return pkgs, nil
}

func (s *PkgClient) GetPackageInfoByDepsdevVersion(ctx context.Context, system, name, version string) (*PackageInfo, error) {
	var depsClient *depsdev.DepsDevClient
	depsClient, err := depsdev.NewClient()
	if err != nil {
		return nil, err
	}
	verItem, err := depsClient.Version(ctx, system, name, version)
	if err != nil {
		return nil, err
	}
	licenses := verItem.GetLicenses()
	links := verItem.GetLinks()
	var homePage string
	var description string
	var repoUrl string
	if len(links) > 0 {
		homePage = links[0].Url
	}

	var projectID string
	projects := verItem.GetRelatedProjects()
	if len(projects) > 0 {
		projectID = projects[0].ProjectKey.Id
	}
	if projectID != "" {
		project, err := depsClient.GetProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		homePage = project.Homepage
		description = project.Description
		if strings.HasPrefix(projectID, "github.com") {
			repoUrl = "https://" + projectID
		}
	}
	return &PackageInfo{
		Name:        name,
		Description: description,
		HomepageURL: homePage,
		RepoURL:     repoUrl,
		Licenses:    licenses,
		Version:     version,
		ProjectID:   projectID,
	}, nil
}

// GetPackageVersions 获取版本列表
func (s *PkgClient) GetPackageVersions(ctx context.Context, system, name string) ([]*PackageInfo, error) {
	var depsClient *depsdev.DepsDevClient
	depsClient, err := depsdev.NewClient()
	if err != nil {
		return nil, err
	}
	vs, err := depsClient.GetPackage(ctx, system, name)
	if err != nil {
		return nil, err
	}
	ls := make([]*PackageInfo, len(vs.Versions))
	for i, v := range vs.Versions {
		ls[i] = FromDepsdevPackageVersion(v)
	}
	return ls, err
}

var DefualtPkgClient = NewPkgClient()

// PurlToPkgInfo 将 PURL 对象转换为 system、name、version
// 根据 PURL 类型规范处理 namespace 和 name 的组合：
// - npm: namespace(scope) + "/" + name, 如 @babel/core
// - golang: namespace + "/" + name, 如 github.com/golang/go
// - maven: namespace + ":" + name, 如 org.apache.maven:maven-core
// - 其他类型: 直接使用 name
func PurlToPkgInfo(p purl.PackageURL) (system, name, version string) {
	system = p.Type
	version = p.Version

	if p.Namespace == "" {
		name = p.Name
		return
	}

	switch p.Type {
	case purl.TypeNPM:
		// npm: @scope/name 格式
		name = p.Namespace + "/" + p.Name
	case purl.TypeGolang:
		version = strings.TrimPrefix(p.Version, "v")
		// golang: full_module_path/name 格式
		name = p.Namespace + "/" + p.Name
	case purl.TypeMaven:
		// maven: groupId:artifactId 格式
		name = p.Namespace + ":" + p.Name
	default:
		// 其他类型: namespace/name 格式
		name = p.Namespace + "/" + p.Name
	}
	return
}

func GetPackageInfoByPurl(ctx context.Context, purlStr string) (pkg *PackageInfo, err error) {
	p, err := purl.FromString(purlStr)
	if err != nil {
		return
	}
	system, name, version := PurlToPkgInfo(p)
	return GetPackageInfo(ctx, system, name, version)
}
func GetPackageInfo(ctx context.Context, system, name, version string) (pkg *PackageInfo, err error) {
	pkg, err = DefualtPkgClient.GetPackageInfo(ctx, system, name, version)
	// 简单使用deps.dev重试3次
	for range 3 {
		if err == nil {
			return
		}
		if version == "" {
			pkg, err = DefualtPkgClient.GetPackageInfo(ctx, system, name, version)
		} else {
			pkg, err = DefualtPkgClient.GetPackageInfoByDepsdevVersion(ctx, system, name, version)
		}
	}
	return
}

func GetPackageVersions(ctx context.Context, system, name string) ([]*PackageInfo, error) {
	return DefualtPkgClient.GetPackageVersions(ctx, system, name)
}

func GetRepositoryInfo(ctx context.Context, repo, name string) (*RepositoryInfo, error) {
	return &RepositoryInfo{}, nil
}
