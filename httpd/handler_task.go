package httpd

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/alayou/techstack/httpd/dao"
	. "github.com/alayou/techstack/httpd/httputil"
	"github.com/alayou/techstack/httpd/taskmgr"
	"github.com/alayou/techstack/model"
	"gorm.io/gorm"
)

// BackgroundTaskWithRelated 任务详情响应（带关联对象）
type BackgroundTaskWithRelated struct {
	model.BackgroundTask
	PublicRepo *model.PublicRepo `json:"public_repo,omitempty"`
	// 预留字段，将来可添加用户仓库、技术栈、脚手架等关联对象
}

// GetBackgroundTask 获取任务详情 GET /api/v1/c/tasks/{id}
func (s *Server) GetBackgroundTask(w http.ResponseWriter, req *http.Request) {
	userID := model.ID(UidGet(req))
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	idStr := req.PathValue("id")
	if idStr == "" {
		Bad(w, "缺少任务ID")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的任务ID")
		return
	}

	var task model.BackgroundTask
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.Where("id = ? AND user_id = ?", id, userID).First(&task).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "任务不存在")
			return
		}
		Bad(w, fmt.Sprintf("查询任务失败: %v", err))
		return
	}

	// 构建响应
	response := BackgroundTaskWithRelated{
		BackgroundTask: task,
	}

	// 根据任务类型关联查询相关对象
	if task.PubRepoID > 0 {
		var pubRepo model.PublicRepo
		_ = dao.View(func(tx *gorm.DB) error {
			return tx.First(&pubRepo, task.PubRepoID).Error
		})
		if pubRepo.ID > 0 {
			response.PublicRepo = &pubRepo
		}
	}

	Ok(w, response)
}

// ListBackgroundTasks 获取任务列表 GET /api/v1/c/tasks
func (s *Server) ListBackgroundTasks(w http.ResponseWriter, req *http.Request) {
	userID := model.ID(UidGet(req))
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	// 分页参数校验
	page, _ := strconv.Atoi(req.FormValue("page"))
	size, _ := strconv.Atoi(req.FormValue("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	} else if size > 100 {
		size = 100
	}

	taskType := req.FormValue("task_type")
	status := req.FormValue("status")

	var tasks = make([]model.BackgroundTask, 0)
	var total int64

	err := dao.View(func(tx *gorm.DB) error {
		// 1. 构建基础查询（仅用于统计总数）
		countDb := tx.Model(&model.BackgroundTask{}).Where("user_id = ?", userID)
		if taskType != "" {
			countDb = countDb.Where("task_type = ?", taskType)
		}
		if status != "" {
			countDb = countDb.Where("status = ?", status)
		}
		if err := countDb.Count(&total).Error; err != nil {
			return err
		}

		// 2. 构建列表查询（分页）
		listDb := tx.Model(&model.BackgroundTask{}).Where("user_id = ?", userID)
		if taskType != "" {
			listDb = listDb.Where("task_type = ?", taskType)
		}
		if status != "" {
			listDb = listDb.Where("status = ?", status)
		}

		return listDb.Order("created_at desc").Offset((page - 1) * size).Limit(size).Find(&tasks).Error
	})

	if err != nil {
		Bad(w, fmt.Sprintf("查询任务失败: %v", err))
		return
	}

	OkList(w, tasks, total)
}

// RetryBackgroundTask 重试失败任务 POST /api/v1/c/tasks/{id}/retry
func (s *Server) RetryBackgroundTask(w http.ResponseWriter, req *http.Request) {
	userID := model.ID(UidGet(req))
	if userID == 0 {
		Bad(w, "未登录")
		return
	}

	idStr := req.PathValue("id")
	if idStr == "" {
		Bad(w, "缺少任务ID")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Bad(w, "无效的任务ID")
		return
	}

	var task model.BackgroundTask
	if err := dao.View(func(tx *gorm.DB) error {
		return tx.Where("id = ? AND user_id = ?", id, userID).First(&task).Error
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			Bad(w, "任务不存在")
			return
		}
		Bad(w, fmt.Sprintf("查询任务失败: %v", err))
		return
	}

	// 状态校验
	if task.Status != model.TaskStatusFailed {
		Bad(w, "只有失败的任务可以重试")
		return
	}
	if task.RetryCount >= task.MaxRetry {
		Bad(w, "已达到最大重试次数")
		return
	}

	// 🔥 修复：重置任务状态（删除冗余代码）
	if err := dao.Transaction(func(tx *gorm.DB) error {
		task.RetryCount++
		return tx.Model(&task).Updates(map[string]interface{}{
			"status":      model.TaskStatusPending,
			"progress":    0,
			"message":     "",
			"retry_count": task.RetryCount,
			"started_at":  nil,
			"finished_at": nil,
		}).Error
	}); err != nil {
		Bad(w, fmt.Sprintf("重置任务失败: %v", err))
		return
	}

	taskmgr.StartTaskAsync(&task)
	Ok(w, task)
}
