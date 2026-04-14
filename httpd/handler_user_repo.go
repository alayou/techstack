package httpd

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/alayou/techstack/httpd/dao"
	. "github.com/alayou/techstack/httpd/httputil"
	"github.com/alayou/techstack/model"
	"gorm.io/gorm"
)

// StarRepo 收藏公共仓库 POST /api/v1/c/public-repos/{id}/star
func (s *Server) StarRepo(w http.ResponseWriter, req *http.Request) {
	userID := UidGet(req)
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	idStr := req.PathValue("id")
	if idStr == "" {
		Bad(w, "缺少仓库ID")
		return
	}

	repoID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的仓库ID")
		return
	}

	// 检查公共仓库是否存在
	var publicRepo model.PublicRepo
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.First(&publicRepo, repoID).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "公共仓库不存在")
			return
		}
		Bad(w, "查询仓库失败: "+err.Error())
		return
	}

	// 检查是否已收藏
	var exist model.UserRepoStar
	_ = dao.View(func(tx *gorm.DB) error {
		return tx.Where("user_id = ? AND public_repo_id = ?", userID, repoID).First(&exist).Error
	})
	if exist.ID != 0 {
		Bad(w, "已收藏该仓库")
		return
	}

	// 创建收藏记录
	star := model.UserRepoStar{
		ID:           model.NewID(),
		UserID:       userID,
		PublicRepoID: repoID,
	}

	if err := dao.Transaction(func(tx *gorm.DB) error {
		return tx.Create(&star).Error
	}); err != nil {
		Bad(w, "收藏失败: "+err.Error())
		return
	}

	Ok(w, star)
}

// UnStarRepo 取消收藏 DELETE /api/v1/c/public-repos/{id}/star
func (s *Server) UnStarRepo(w http.ResponseWriter, req *http.Request) {
	userID := UidGet(req)
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	idStr := req.PathValue("id")
	if idStr == "" {
		Bad(w, "缺少仓库ID")
		return
	}

	repoID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的仓库ID")
		return
	}

	// 检查收藏记录是否存在
	var star model.UserRepoStar
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.Where("user_id = ? AND public_repo_id = ?", userID, repoID).First(&star).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "未收藏该仓库")
			return
		}
		Bad(w, "查询收藏记录失败: "+err.Error())
		return
	}

	// 删除收藏记录
	if err := dao.Transaction(func(tx *gorm.DB) error {
		return tx.Delete(&star).Error
	}); err != nil {
		Bad(w, "取消收藏失败: "+err.Error())
		return
	}

	Ok(w, nil)
}

// ListUserStarRepos 获取用户收藏列表 GET /api/v1/c/user/stars
func (s *Server) ListUserStarRepos(w http.ResponseWriter, req *http.Request) {
	userID := UidGet(req)
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	// 分页参数
	page, size := ParsePageParams(req)

	// 查询收藏记录及关联的公共仓库
	var stars []model.UserRepoStar
	var total int64

	err := dao.View(func(tx *gorm.DB) error {
		// 查询总数
		if err := tx.Model(&model.UserRepoStar{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
			return err
		}

		// 查询列表
		return tx.Where("user_id = ?", userID).
			Order("created_at desc").
			Offset((page - 1) * size).
			Limit(size).
			Find(&stars).Error
	})

	if err != nil {
		Bad(w, "查询收藏失败: "+err.Error())
		return
	}

	// 填充关联的公共仓库信息
	type StarWithRepo struct {
		model.UserRepoStar
		PublicRepo model.PublicRepo `json:"public_repo"`
	}

	var result = make([]StarWithRepo, 0)
	for _, star := range stars {
		var publicRepo model.PublicRepo
		_ = dao.View(func(tx *gorm.DB) error {
			return tx.First(&publicRepo, star.PublicRepoID).Error
		})
		result = append(result, StarWithRepo{
			UserRepoStar: star,
			PublicRepo:   publicRepo,
		})
	}

	OkList(w, result, total)
}

// ==============================
// 工具函数
// ==============================

// extractRepoName 从URL中提取仓库名称
func extractRepoName(url string) string {
	// 处理常见的 Git URL 格式
	// github.com/owner/repo -> repo
	// git@github.com:owner/repo -> repo
	// https://github.com/owner/repo.git -> repo

	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// extractProvider 从URL中提取仓库平台
func extractProvider(url string) string {
	lowerURL := strings.ToLower(url)

	if strings.Contains(lowerURL, "github.com") {
		return "github"
	}
	if strings.Contains(lowerURL, "gitlab.com") || strings.Contains(lowerURL, "gitlab") {
		return "gitlab"
	}
	if strings.Contains(lowerURL, "gitee.com") || strings.Contains(lowerURL, "gitee") {
		return "gitee"
	}
	if strings.Contains(lowerURL, "bitbucket.org") {
		return "bitbucket"
	}
	return "other"
}
