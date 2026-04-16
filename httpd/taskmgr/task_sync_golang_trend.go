package taskmgr

import (
	"context"
	"fmt"
	"time"

	"github.com/alayou/techstack/httpd/dao"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/pkg/pkgclient/gogo/goproxycn"
	"gorm.io/gorm"
)

// executeSyncGolangTrendTask 执行同步 golang 包趋势任务
// 从 goproxy.cn 获取最近一段时间内最活跃的 TOP 1000 模块列表，并同步到数据库
// 智能选择趋势类型：
// - 检查7天内是否有 last-7-days 同步成功的任务，没有则使用 last-7-days
// - 检查30天内是否有 last-30-days 同步成功的任务，没有则使用 last-30-days
// - 否则使用 latest
func executeSyncGolangTrendTask(task *model.BackgroundTask) (err error) {
	ctx := context.Background()

	// 更新任务状态：开始获取趋势数据
	err = dao.UpdateTaskStatus(task.ID, 10, model.TaskStatusRunning, "开始从 goproxy.cn 获取趋势数据...")
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 智能选择趋势类型
	var trendType goproxycn.TrendType
	var trendTypeDesc string

	// 检查7天内是否有 last-7-days 同步成功的任务
	if !dao.HasTrendTypeSyncedInDays(TaskTypeSyncGolangTrend, "last-7-days", 7) {
		// 7天内没有 last-7-days 同步成功，使用 last-7-days
		trendType = goproxycn.TrendLast7Days
		trendTypeDesc = "最近7天趋势"
	} else if !dao.HasTrendTypeSyncedInDays(TaskTypeSyncGolangTrend, "last-30-days", 30) {
		// 7天内有 last-7-days 同步成功，但30天内没有 last-30-days 同步成功，使用 last-30-days
		trendType = goproxycn.TrendLast30Days
		trendTypeDesc = "最近30天趋势"
	} else {
		// 7天内有 last-7-days 同步成功，且30天内也有 last-30-days 同步成功，使用 latest
		trendType = goproxycn.TrendLatest
		trendTypeDesc = "最新趋势"
	}

	err = dao.UpdateTaskStatus(task.ID, 15, model.TaskStatusRunning,
		fmt.Sprintf("使用趋势类型: %s", trendTypeDesc))
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 获取趋势数据
	trendItems, err := goproxycn.GetStatsTrends(ctx, trendType)
	if err != nil {
		return fmt.Errorf("获取趋势数据失败: %v", err)
	}

	totalCount := len(trendItems)
	if totalCount == 0 {
		return fmt.Errorf("未获取到趋势数据")
	}

	err = dao.UpdateTaskStatus(task.ID, 20, model.TaskStatusRunning, fmt.Sprintf("成功获取 %d 个模块趋势数据，开始同步到数据库...", totalCount))
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 同步数据到数据库
	successCount := 0
	failedCount := 0

	for i, item := range trendItems {
		progress := 20 + int(float64(i)/float64(totalCount)*70) // 20% -> 90%

		// 检查包是否已存在
		var existingPkg model.Package
		err = dao.View(func(tx *gorm.DB) error {
			return tx.Where("purl_type = ? AND name = ?", "golang", item.ModulePath).
				First(&existingPkg).Error
		})

		if err == nil {
			// 包已存在，更新时间戳
			err = dao.Transaction(func(tx *gorm.DB) error {
				return tx.Model(&existingPkg).Updates(map[string]interface{}{
					"updated_at": time.Now().Unix(),
				}).Error
			})
			if err != nil {
				failedCount++
				continue
			}
			successCount++
		} else if err == gorm.ErrRecordNotFound {
			// 包不存在，创建新包
			newPkg := &model.Package{
				Name:           item.ModulePath,
				PurlType:       "golang",
				NormalizedName: normalizePackageName(item.ModulePath),
				CreatedAt:      time.Now().Unix(),
				UpdatedAt:      time.Now().Unix(),
			}

			err = dao.Transaction(func(tx *gorm.DB) error {
				return tx.Create(newPkg).Error
			})
			if err != nil {
				failedCount++
				continue
			}
			successCount++
		} else {
			// 其他错误
			failedCount++
			continue
		}

		// 每处理 50 个包更新一次进度
		if (i+1)%50 == 0 || i == totalCount-1 {
			err = dao.UpdateTaskStatus(task.ID, progress, model.TaskStatusRunning,
				fmt.Sprintf("已处理 %d/%d 个模块 (成功: %d, 失败: %d)", i+1, totalCount, successCount, failedCount))
			if err != nil {
				return fmt.Errorf("更新任务状态失败: %v", err)
			}
		}
	}

	err = dao.UpdateTaskStatus(task.ID, 100, model.TaskStatusRunning,
		fmt.Sprintf("同步完成！使用趋势类型: %s，总计: %d, 成功: %d, 失败: %d", trendTypeDesc, totalCount, successCount, failedCount))
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	return nil
}

// normalizePackageName 标准化包名称（统一小写）
func normalizePackageName(name string) string {
	// 简单实现：将包名转换为小写
	// 可以根据需要添加更复杂的标准化逻辑
	result := ""
	for _, c := range name {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

// CreateSyncTrendTask 创建同步趋势任务
func CreateSyncTrendTask(userID int64) (*model.BackgroundTask, error) {
	task := &model.BackgroundTask{
		UserID:     userID,
		TaskType:   TaskTypeSyncGolangTrend,
		Status:     model.TaskStatusPending,
		Progress:   0,
		RetryCount: 0,
		MaxRetry:   3,
	}

	// 入库
	if err := dao.Transaction(func(tx *gorm.DB) error {
		return tx.Create(task).Error
	}); err != nil {
		return nil, err
	}

	startTaskAsync(task)
	return task, nil
}
