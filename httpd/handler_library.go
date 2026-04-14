package httpd

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alayou/techstack/httpd/bind"
	"github.com/alayou/techstack/httpd/dao"
	. "github.com/alayou/techstack/httpd/httputil"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/pkg/pkgclient"
	"github.com/alayou/techstack/pkg/pkgclient/depsdev"
	"gorm.io/gorm"
)

// GetLibraries 分页获取所有库列表
func (s *Server) SearchLibraries(w http.ResponseWriter, req *http.Request) {
	var query bind.QueryPage
	if err := ShouldJson(req, &query); err != nil {
		Bad(w, err)
		return
	}

	// 设置默认值
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Size < 1 {
		query.Size = 20
	}
	if query.Size > 100 {
		query.Size = 100
	}

	// 获取查询参数
	keyword := req.FormValue("keyword")
	purl_type := req.FormValue("purl_type")
	language := req.FormValue("language")

	// 构建查询
	q := dao.NewQuery()
	q.WithPage(query.Page, query.Size)

	// 支持按关键词搜索包名或描述
	if keyword != "" {
		q.WithLike("normalized_name", keyword)
	}

	// 按生态系统过滤
	if purl_type != "" {
		q.WithEq("purl_type", purl_type)
	}

	// 按编程语言过滤
	if language != "" {
		q.WithEq("language", language)
	}

	// 排序
	order := "created_at desc"
	if query.Sort != "" {
		if query.Order == "desc" {
			order = query.Sort + " desc"
		} else {
			order = query.Sort + " asc"
		}
	}

	// 查询数据
	var libraries = make([]model.Package, 0)
	var total int64

	_ = dao.View(func(tx *gorm.DB) error {
		// 查询列表
		if err := tx.Model(&model.Package{}).Where(q.Query(), q.Params()...).Order(order).Offset(q.Offset).Limit(q.Limit).Find(&libraries).Error; err != nil {
			return err
		}
		// 查询总数
		return tx.Model(&model.Package{}).Where(q.Query(), q.Params()...).Count(&total).Error
	})

	OkList(w, libraries, total)
}

// GetPackage 获取库详情,根据Purl或者purlType+uniqueName
// 路由: GET /api/v1/c/libraries/get/purl?purl=xxx 或 GET /api/v1/c/libraries/get/purl?purl_type=xxx&name=xxx
func (s *Server) GetPackage(w http.ResponseWriter, req *http.Request) {
	// 获取查询参数
	purlStr := req.FormValue("purl")
	purlType := req.FormValue("purl_type")
	name := req.FormValue("name")

	var pkgmod model.Package
	var err error

	if purlStr != "" {
		// ==============================
		// 1. 通过 PURL 查询
		// ==============================
		// PURL 格式: pkgtype:name[@version] 例如: npm:lodash 或 npm:lodash@4.0.0
		pkgmod, err = getPackageByPurl(purlStr)
		if err != nil {
			Bad(w, err.Error())
			return
		}
	} else if purlType != "" && name != "" {
		// ==============================
		// 2. 通过 purl_type + name 查询
		// ==============================
		normalizedName := strings.ToLower(name)
		err = dao.View(func(tx *gorm.DB) error {
			return tx.Where("normalized_name = ? AND purl_type = ?", normalizedName, purlType).First(&pkgmod).Error
		})
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				Bad(w, "库不存在")
				return
			}
			Bad(w, "查询库失败: "+err.Error())
			return
		}
	} else {
		Bad(w, "请提供 purl 参数或 purl_type + name 参数")
		return
	}

	Ok(w, pkgmod)
}

// getPackageByPurl 通过 PURL 解析并查询库
// PURL 格式: pkgtype:name[@version] 例如: npm:lodash 或 npm:lodash@4.0.0
func getPackageByPurl(purlStr string) (model.Package, error) {
	var pkgmod model.Package

	// 解析 PURL 格式: pkgtype:name[@version]
	// 首先去掉版本部分（如果有）
	name := purlStr
	purlType := ""

	// 检查是否有 @version 后缀
	if atIdx := strings.LastIndex(purlStr, "@"); atIdx > 0 {
		// 确保 @ 不是在冒号之前（因为 PURL 是 pkgtype:name@version）
		colonIdx := strings.Index(purlStr, ":")
		if colonIdx >= 0 && atIdx > colonIdx {
			name = purlStr[:atIdx]
		}
	}

	// 解析 pkgtype:name 部分
	colonIdx := strings.Index(name, ":")
	if colonIdx < 0 {
		return pkgmod, errors.New("无效的 PURL 格式，应为 pkgtype:name 或 pkgtype:name@version")
	}
	purlType = name[:colonIdx]
	name = name[colonIdx+1:]

	if purlType == "" || name == "" {
		return pkgmod, errors.New("无效的 PURL 格式")
	}

	// 标准化包名
	normalizedName := strings.ToLower(name)

	// 查询数据库
	err := dao.View(func(tx *gorm.DB) error {
		return tx.Where("normalized_name = ? AND purl_type = ?", normalizedName, purlType).First(&pkgmod).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return pkgmod, errors.New("库不存在")
		}
		return pkgmod, err
	}

	return pkgmod, nil
}

// GetPackageDetail 获取库详情,根据ID
func (s *Server) GetPackageDetail(w http.ResponseWriter, req *http.Request) {
	idStr := req.PathValue("package_id")
	if idStr == "" {
		Bad(w, "缺少库ID")
		return
	}
	packageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的库ID")
		return
	}

	var pkgmod model.Package
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.First(&pkgmod, packageID).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "库不存在")
			return
		}
		Bad(w, "查询库失败: "+err.Error())
		return
	}

	Ok(w, pkgmod)
}

// SyncLiraryRequest 同步库的请求参数
type SyncLiraryRequest struct {
	// ID 库ID
	ID string `json:"id" validate:"required"`
}

// AddFromRepoLiraryRequest 从项目添加库的请求参数
type AddFromRepoLiraryRequest struct {
	// Content 项目依赖文件内容
	Content string `json:"content" validate:"required"`
	// FileType 文件类型: go.mod, package.json, uv.lock, requirements.txt
	FileType string `json:"fileType" validate:"required"`
}

// ManualAddPackage 手动添加全局第三方依赖库
// 路由: POST /api/v1/c/libraries
// 需登录鉴权: checkUserToken 中间件
func (s *Server) ManualAddPackage(w http.ResponseWriter, req *http.Request) {
	// ==============================
	// 1. 解析并校验 JSON 请求体
	// ==============================
	var query bind.CreatePackageRequest
	if err := ShouldJson(req, &query); err != nil {
		Bad(w, err)
		return
	}

	// ==============================
	// 2. 参数校验
	// ==============================
	// name 不能为空
	if strings.TrimSpace(query.Name) == "" {
		Bad(w, "包名不能为空")
		return
	}

	// PurlType 必须是支持的类型：npm/cargo/pypi/golang
	validPurlTypes := map[string]bool{
		"npm":    true,
		"cargo":  true,
		"pypi":   true,
		"golang": true,
	}
	if !validPurlTypes[query.PurlType] {
		Bad(w, "无效的生态系统类型，仅支持: npm, cargo, pypi, golang")
		return
	}

	// ==============================
	// 3. 官方数据源校验包是否真实存在
	// ==============================
	// 使用 pkgclient 校验包是否存在
	packageInfo, err := pkgclient.GetPackageInfo(context.Background(), query.PurlType, query.Name, "")
	if err != nil {
		// 如果是 ErrPackageNotFound，说明包不存在
		if err == pkgclient.ErrPackageNotFound {
			Bad(w, "该包在官方源中不存在")
			return
		}
		Bad(w, "校验包存在性失败: "+err.Error())
		return
	}

	// ==============================
	// 4. 自动补全信息（用户不传则从官方拉取）
	// ==============================
	description := query.Description
	homepageURL := query.HomepageURL
	repoURL := query.RepoURL

	// 如果用户未提供详细信息，从官方源获取
	if packageInfo != nil {
		if description == "" && packageInfo.Description != "" {
			description = packageInfo.Description
		}
		if homepageURL == "" && packageInfo.HomepageURL != "" {
			homepageURL = packageInfo.HomepageURL
		}
		if repoURL == "" && packageInfo.RepoURL != "" {
			repoURL = packageInfo.RepoURL
		}
	}

	// ==============================
	// 5. 本地数据库查重
	// ==============================
	normalizedName := strings.ToLower(query.Name)
	var existing model.Package
	err = dao.View(func(tx *gorm.DB) error {
		return tx.Where("normalized_name = ? AND purl_type = ?", normalizedName, query.PurlType).First(&existing).Error
	})
	if err != nil && err != gorm.ErrRecordNotFound {
		Bad(w, "查询库失败: "+err.Error())
		return
	}
	if err == nil {
		// 记录已存在
		Bad(w, "该生态下的包已存在")
		return
	}
	// ==============================
	// 6. 写入 Package 表（只写入库信息，不写入版本）
	// ==============================
	pkgmod := model.Package{
		Name:           query.Name,
		PurlType:       query.PurlType,
		NormalizedName: normalizedName,
		Description:    description,
		HomepageURL:    homepageURL,
		RepositoryURL:  repoURL,
	}

	if err := dao.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&pkgmod).Error
	}); err != nil {
		Bad(w, "创建库失败: "+err.Error())
		return
	}

	Ok(w, pkgmod)
}

func (s *Server) BatchImportPackage(w http.ResponseWriter, req *http.Request) {
	var query bind.BatchImportPackageRequest
	if err := ShouldJson(req, &query); err != nil {
		Bad(w, err)
		return
	}
	if len(query.List) > 5000 {
		Bad(w, "最大5000")
		return
	}
	if len(query.List) == 0 {
		Ok(w)
		return
	}
	var exist []struct {
		Name     string
		PurlType string
	}
	err := dao.View(func(tx *gorm.DB) error {
		var cond [][]any
		for _, v := range query.List {
			cond = append(cond, []any{v.Name, v.PurlType})
		}
		tx.Model(&model.Package{}).
			Where("(name, purl_type) IN ?", cond).
			Select("name, purl_type").
			Find(&exist)
		return nil
	})
	if err != nil {
		Bad(w, err)
		return
	}
	notexist := make([]depsdev.VersionBatchReq, len(query.List)-len(exist))
	var i int
	for _, pkg := range query.List {
		ok := false
		for _, indb := range exist {
			if pkg.Name == indb.Name && pkg.PurlType == indb.PurlType {
				ok = true
				break
			}
		}
		if ok {
			continue
		}
		notexist[i] = depsdev.VersionBatchReq{
			Name:    pkg.Name,
			System:  pkg.PurlType,
			Version: pkg.Version,
		}
		i += 1
	}

	notexist = notexist[:i]
	if len(notexist) == 0 {
		Ok(w, map[string]any{
			"rows": len(query.List),
		})
		return
	}
	client := pkgclient.NewPkgClient()
	pkgs, err := client.GetPackageInfoListByDepsdev(context.TODO(), notexist)
	if err != nil {
		Bad(w, err)
		return
	}
	if len(pkgs) == 0 {
		Ok(w, map[string]any{
			"rows": 0,
		})
		return
	}
	libs := make([]*model.Package, len(pkgs))
	for i := range libs {
		libs[i] = PackageInfo2LIbrary(pkgs[i])
	}
	var total int64
	err = dao.Transaction(func(tx *gorm.DB) error {
		res := tx.Create(libs)
		total = res.RowsAffected
		return res.Error
	})
	if err != nil {
		Bad(w, err)
		return
	}
	Ok(w, map[string]any{
		"rows": total,
	})
}

// PackageInfo2LIbrary 将官方源获取的包信息转换为数据库模型库
func PackageInfo2LIbrary(info *pkgclient.PackageInfo) *model.Package {
	// 从 PURL 中提取生态系统类型
	purl := info.ToPURL()
	return &model.Package{
		ID:             model.NewID(),
		Name:           info.Name,
		PurlType:       purl.Type,
		NormalizedName: strings.ToLower(info.Name),
		Description:    info.Description,
		HomepageURL:    info.HomepageURL,
		RepositoryURL:  info.RepoURL,
	}
}

// extractPurlType 从 PURL 中提取生态系统类型
func extractPurlType(purl string) string {
	if purl == "" {
		return ""
	}
	// PURL 格式: pkgtype:name/version 例如: npm:lodash/1.0.0
	parts := strings.SplitN(purl, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// PackageInfo 官方源获取的包信息
type PackageInfo struct {
	Description   string
	HomepageURL   string
	RepositoryURL string
}

// SyncPackageVersionsRequest 同步库版本的请求参数
type SyncPackageVersionsRequest struct {
	SyncStrategy string `json:"sync_strategy" validate:"required,oneof=all"`
}

// SyncPackageVersionsResponse 同步库版本的响应
type SyncPackageVersionsResponse struct {
	PackageID int64    `json:"package_id,string"`
	Strategy  string   `json:"sync_strategy"`
	Total     int      `json:"total"`
	Added     int      `json:"added"`
	Skipped   int      `json:"skipped"`
	Versions  []string `json:"versions"`
}

// SyncPackageVersions 按策略同步官方版本到 PackageVersion 表
// 路由: POST /api/v1/c/libraries/{package_id}/sync-versions
// 需登录鉴权: checkUserToken 中间件
func (s *Server) SyncPackageVersions(w http.ResponseWriter, req *http.Request) {
	idStr := req.PathValue("package_id")
	if idStr == "" {
		Bad(w, "缺少库ID")
		return
	}
	packageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的库ID")
		return
	}

	// ==============================
	// 查询 Package 表是否存在该库
	// ==============================
	var pkgmod model.Package
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.First(&pkgmod, packageID).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "库不存在")
			return
		}
		Bad(w, "查询库失败: "+err.Error())
		return
	}

	// ==============================
	// 读取请求体策略
	// ==============================
	var query SyncPackageVersionsRequest
	if err := ShouldJson(req, &query); err != nil {
		Bad(w, err)
		return
	}

	// ==============================
	// 调用 pkgclient.GetPackageVersions 获取官方版本列表
	// ==============================
	versions, err := pkgclient.GetPackageVersions(context.Background(), pkgmod.PurlType, pkgmod.Name)
	if err != nil {
		Bad(w, "获取官方版本列表失败: "+err.Error())
		return
	}

	// ==============================
	// 6. 在事务内插入版本（不重复插入）
	// ==============================
	var addedCount int
	var skippedCount int
	var versionStrings []string

	if err := dao.Transaction(func(tx *gorm.DB) error {
		for _, v := range versions {
			// 记录版本字符串用于返回
			versionStrings = append(versionStrings, v.Version)

			// 检查版本是否已存在
			var existing model.PackageVersion
			err := tx.Where("package_id = ? AND version = ?", packageID, v.Version).First(&existing).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err == nil {
				// 已存在，跳过
				skippedCount++
				continue
			}
			// 插入新版本
			version := model.PackageVersion{
				ID:             model.NewID(),
				PackageID:      packageID,
				Version:        v.Version,
				PURL:           v.ToPURL().String(),
				PublishedAt:    v.ReleasedAt.Unix(),
				PublishedAtStr: v.ReleasedAt.Format(time.DateTime),
			}
			if err := tx.Create(&version).Error; err != nil {
				return err
			}
			addedCount++
		}
		return nil
	}); err != nil {
		Bad(w, "同步版本失败: "+err.Error())
		return
	}

	response := SyncPackageVersionsResponse{
		PackageID: packageID,
		Strategy:  query.SyncStrategy,
		Total:     len(versions),
		Added:     addedCount,
		Skipped:   skippedCount,
		Versions:  versionStrings,
	}

	Ok(w, response)
}

// GetPackageVersions 获取某个库的已同步版本列表（从 PackageVersion 表）
func (s *Server) GetPackageVersions(w http.ResponseWriter, req *http.Request) {
	idStr := req.PathValue("package_id")
	if idStr == "" {
		Bad(w, "缺少库ID")
		return
	}

	packageID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的库ID格式")
		return
	}

	// 先检查库是否存在
	var pkgmod model.Package
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.First(&pkgmod, "id = ?", packageID).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "库不存在")
			return
		}
		Bad(w, "查询库失败: "+err.Error())
		return
	}

	// 查询该库的所有版本，按发布时间倒序
	var versions = make([]model.PackageVersion, 0)
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.Where("package_id = ?", packageID).
			Order("published_at DESC NULLS LAST, version DESC").
			Find(&versions).Error
	}); err != nil {
		Bad(w, "查询版本失败: "+err.Error())
		return
	}

	OkList(w, versions, int64(len(versions)))
}
