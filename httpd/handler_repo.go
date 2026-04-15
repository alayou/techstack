package httpd

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/alayou/techstack/httpd/dao"
	. "github.com/alayou/techstack/httpd/httputil"
	"github.com/alayou/techstack/httpd/taskmgr"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/pkg/pkgclient/gogithub"
	"gorm.io/gorm"
)

const (
	PublicRepoType = "public"

	// 排序白名单（防SQL注入）
	AllowedSortFields = "repo_name,stars,created_at,updated_at"
)

// PublicRepoWithStar 公共仓库+收藏状态
type PublicRepoWithStar struct {
	model.PublicRepo
	IsStarred bool `json:"is_starred"`
}

// ListPublicRepos 获取公共仓库列表
// 路由: GET /api/v1/c/public-repos
// 鉴权: 登录用户
// 参数:
//   - keyword: 仓库名称关键字
//   - language: 编程语言
//   - sort: 排序字段 (name,stars,language,created_at,updated_at)
//   - order: 排序方向 (asc,desc)
//   - starred: 是否收藏过滤 (true,false，不传则不过滤)
func (s *Server) ListPublicRepos(w http.ResponseWriter, req *http.Request) {
	userID := UidGet(req)
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	// 解析分页参数
	page, size := ParsePageParams(req)

	// 解析查询参数
	keyword := strings.TrimSpace(req.FormValue("keyword"))
	language := strings.TrimSpace(req.FormValue("language"))
	sort := strings.TrimSpace(req.FormValue("sort"))
	orderParam := strings.TrimSpace(req.FormValue("order"))
	starredParam := strings.TrimSpace(req.FormValue("starred"))

	// 构建查询
	q := dao.NewQuery()
	q.WithPage(page, size)

	// 条件拼接
	if keyword != "" {
		q.WithLike("repo_name", keyword)
	}
	if language != "" {
		q.WithEq("language", language)
	}

	order := "stars desc, created_at desc"
	if sort != "" && slices.Contains(strings.Split(AllowedSortFields, ","), sort) {
		if orderParam == "asc" {
			order = sort + " asc"
		} else {
			order = sort + " desc"
		}
	}

	// 连表查询：LEFT JOIN t_user_stars 获取用户收藏状态
	var result []PublicRepoWithStar
	var total int64

	err := dao.View(func(tx *gorm.DB) error {
		// 主查询：LEFT JOIN 获取仓库列表及收藏状态
		query := tx.Table(model.PublicRepoTableName+" as p").
			Select("p.*, CASE WHEN s.id IS NOT NULL THEN true ELSE false END as is_starred").
			Joins("LEFT JOIN t_user_stars s ON p.id = s.public_repo_id AND s.user_id = ?", userID).
			Where(q.Query(), q.Params()...)

		// starred 参数过滤：只查询已收藏或未收藏的仓库
		if starredParam == "true" {
			query = query.Where("s.id IS NOT NULL")
		} else if starredParam == "false" {
			query = query.Where("s.id IS NULL")
		}

		// 查询总数
		countQuery := tx.Table(model.PublicRepoTableName+" as p").
			Joins("LEFT JOIN t_user_stars s ON p.id = s.public_repo_id AND s.user_id = ?", userID).
			Where(q.Query(), q.Params()...)
		if starredParam == "true" {
			countQuery = countQuery.Where("s.id IS NOT NULL")
		} else if starredParam == "false" {
			countQuery = countQuery.Where("s.id IS NULL")
		}
		if err := countQuery.Count(&total).Error; err != nil {
			return err
		}

		// 查询列表
		return query.Order(order).Offset(q.Offset).Limit(q.Limit).Find(&result).Error
	})
	if err != nil {
		Bad(w, "查询公共仓库失败")
		return
	}

	OkList(w, result, total)
}

// ImportPublicRepoRequest 导入公共仓库请求
type ImportPublicRepoRequest struct {
	RepoURL     string `json:"repo_url" validate:"required,url"`
	Description string `json:"description"`
}

// ImportPublicRepo 导入公共仓库到系统
// 路由: POST /api/v1/c/public-repos/import
// 鉴权: 登录 + 管理员权限
func (s *Server) ImportPublicRepo(w http.ResponseWriter, req *http.Request) {
	userID := UidGet(req)
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	// if !IsAdmin(userID) {
	// 	Forbidden(w, "无权限操作")
	// 	return
	// }

	// 参数解析 + 完整校验
	var query ImportPublicRepoRequest
	if err := ShouldJson(req, &query); err != nil {
		Bad(w, "参数解析失败："+err.Error())
		return
	}
	// 手动校验必填（兼容validate）
	if query.RepoURL == "" {
		Bad(w, "仓库地址不能为空")
		return
	}

	// 查重：仓库已存在
	var existing model.PublicRepo
	err := dao.View(func(tx *gorm.DB) error {
		return tx.Where("repo_url = ?", query.RepoURL).First(&existing).Error
	})
	if err == nil {
		Bad(w, "该仓库已存在")
		return
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		Bad(w, "查询仓库失败")
		return
	}

	// 调用 GitHub API 获取仓库信息
	repoURL := strings.ToLower(query.RepoURL)
	repoURL = strings.TrimSuffix(repoURL, ".git")
	if !strings.Contains(repoURL, "github.com") {
		Bad(w, "仅支持 GitHub 仓库导入")
		return
	}

	ctx := context.Background()
	repoInfo, err := gogithub.GetGithubRepoInfo(ctx, query.RepoURL)
	if err != nil {
		if err == gogithub.ErrRepoNotFound {
			Bad(w, "仓库不存在或无法访问")
			return
		}
		if err == gogithub.ErrorRepoURLInvalid {
			Bad(w, "仓库地址格式无效")
			return
		}
		Bad(w, "获取仓库信息失败："+err.Error())
		return
	}
	if repoInfo.DefaultBranch == "" {
		repoInfo.DefaultBranch = "main"
	}
	if repoInfo.Description != "" {
		query.Description = repoInfo.Description
	}
	// 创建公共仓库记录
	publicRepo := model.PublicRepo{
		FullName:       repoInfo.FullName,
		RepoURL:        query.RepoURL,
		RepoName:       repoInfo.Name,
		DefaultBranch:  repoInfo.DefaultBranch,
		Description:    query.Description,
		Language:       repoInfo.Language,
		Stars:          int64(repoInfo.Stars),
		Forks:          int64(repoInfo.Forks),
		License:        repoInfo.License,
		ImportStatus:   model.RepoStatusPending,
		AnalysisStatus: model.RepoStatusWaiting,
	}

	// 事务入库
	if err := dao.Repo.Create(&publicRepo); err != nil {
		Bad(w, "创建仓库失败")
		return
	}

	_, _ = taskmgr.CreateTask(userID, taskmgr.TaskTypeRepoImport, publicRepo.ID.Int64())

	Ok(w, publicRepo)
}

// PublicRepoDetailResponse 公共仓库详情响应
type PublicRepoDetailResponse struct {
	model.PublicRepo
	Dependencies  []RepoDependencyDetail `json:"dependencies"`
	TechAnanlysis model.RepoTechAnalysis `json:"tech_analysis"`
}

// RepoDependencyDetail 依赖详情（带是否纳入标记）
type RepoDependencyDetail struct {
	model.RepoDependency
	PkgID        int64 `json:"pkg_id,string,omitempty"`
	PkgVersionID int64 `json:"pkg_version_id,string,omitempty"`
}

// GetPublicRepo 获取公共仓库详情
// 路由: GET /api/v1/c/public-repos/{id}
// 鉴权: 登录用户
func (s *Server) GetPublicRepo(w http.ResponseWriter, req *http.Request) {
	if UidGet(req) == 0 {
		Bad(w, "未登录")
		return
	}

	idStr := req.PathValue("id")
	repoID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的仓库ID")
		return
	}

	var repo model.PublicRepo
	err = dao.View(func(tx *gorm.DB) error {
		return tx.First(&repo, repoID).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "仓库不存在")
			return
		}
		Bad(w, "查询仓库失败")
		return
	}

	// 关联查询技术分析
	var techAnalysis model.RepoTechAnalysis
	_ = dao.View(func(tx *gorm.DB) error {
		return tx.Where("repo_type = ? AND repo_id = ?", PublicRepoType, repoID).
			First(&techAnalysis).Error
	})

	// 关联查询依赖库
	var dependencies = make([]model.RepoDependency, 0)
	err = dao.View(func(tx *gorm.DB) error {
		return tx.Where("repo_type = ? AND repo_id = ?", PublicRepoType, repoID).
			Order("source_file, version").
			Find(&dependencies).Error
	})
	if err != nil {
		Bad(w, "查询依赖库失败")
		return
	}

	// 批量查询已纳入的依赖ID
	includedDepIDs := make(map[int64]int64)
	includedDepVersionIDs := make(map[int64]int64)
	_ = dao.View(func(tx *gorm.DB) error {
		// 查询 RepoPkgIndex 中已纳入的依赖
		var pkgIndexes []model.RepoPkgIndex
		if err := tx.Where("repo_id = ?", repoID).Find(&pkgIndexes).Error; err != nil {
			return err
		}
		for _, idx := range pkgIndexes {
			includedDepIDs[idx.DepID] = idx.PkgID
		}

		// 查询 RepoPkgVersionIndex 中已纳入的依赖
		var versionIndexes []model.RepoPkgVersionIndex
		if err := tx.Where("repo_id = ?", repoID).Find(&versionIndexes).Error; err != nil {
			return err
		}
		for _, idx := range versionIndexes {
			includedDepVersionIDs[idx.DepID] = idx.PkgID
		}
		return nil
	})

	// 组装依赖详情（带纳入标记）
	dependencyDetails := make([]RepoDependencyDetail, 0, len(dependencies))
	for _, dep := range dependencies {
		detail := RepoDependencyDetail{
			RepoDependency: dep,
			PkgID:          includedDepIDs[dep.ID.Int64()],
			PkgVersionID:   includedDepVersionIDs[dep.ID.Int64()],
		}
		dependencyDetails = append(dependencyDetails, detail)
	}

	// 构建响应
	response := PublicRepoDetailResponse{
		PublicRepo:    repo,
		Dependencies:  dependencyDetails,
		TechAnanlysis: techAnalysis,
	}

	Ok(w, response)
}

// AnalysisPublicRepo 根据仓库状态创建导入或分析任务
// 路由: GET /api/v1/c/public-repos/{id}/analysis
// 鉴权: 登录用户
// 条件:
//   - import_status 为 RepoStatusWaiting 时，创建导入任务
//   - import_status 为 RepoStatusSuccess 且 analysis_status 为 RepoStatusWaiting 时，创建分析任务
func (s *Server) AnalysisPublicRepo(w http.ResponseWriter, req *http.Request) {
	userID := UidGet(req)
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	idStr := req.PathValue("id")
	repoID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的仓库ID")
		return
	}

	// 查询仓库信息
	var repo model.PublicRepo
	err = dao.View(func(tx *gorm.DB) error {
		return tx.First(&repo, repoID).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "仓库不存在")
			return
		}
		Bad(w, "查询仓库失败")
		return
	}

	// import_status 为 waiting 时，优先创建导入任务
	if repo.ImportStatus == model.RepoStatusWaiting {
		if err := dao.Repo.UpdateRepoImportStatus(repoID, model.RepoStatusPending); err != nil {
			Bad(w, "更新导入状态失败")
			return
		}

		task, err := taskmgr.CreateTask(userID, taskmgr.TaskTypeRepoImport, repoID)
		if err != nil {
			_ = dao.Repo.UpdateRepoImportStatus(repoID, model.RepoStatusWaiting)
			Bad(w, "创建导入任务失败")
			return
		}

		Ok(w, task)
		return
	}

	// 导入未完成时，不允许继续创建分析任务
	if repo.ImportStatus != model.RepoStatusSuccess {
		Bad(w, "仓库导入未完成，当前状态无法创建任务")
		return
	}

	// analysis_status 为 waiting 时，创建分析任务
	if repo.AnalysisStatus != model.RepoStatusWaiting {
		Bad(w, "当前状态不允许创建任务")
		return
	}

	if err := dao.Repo.UpdateRepoAnalysisStatus(repoID, model.RepoStatusPending); err != nil {
		Bad(w, "更新分析状态失败")
		return
	}

	task, err := taskmgr.CreateRepoAnalysisTask(userID, taskmgr.TaskTypeAnalysisPublicRepo, repoID, "", nil)
	if err != nil {
		_ = dao.Repo.UpdateRepoAnalysisStatus(repoID, model.RepoStatusWaiting)
		Bad(w, "创建分析任务失败")
		return
	}

	Ok(w, task)
}

// ==============================
// 工具函数：复用分页解析
// ==============================
func ParsePageParams(req *http.Request) (page int, size int) {
	page, _ = strconv.Atoi(req.FormValue("page"))
	size, _ = strconv.Atoi(req.FormValue("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	} else if size > 100 {
		size = 100
	}
	return
}
