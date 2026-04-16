package dao

import (
	"time"

	"github.com/alayou/techstack/model"
	"gorm.io/gorm"
)

func GetLastStopTask() *model.BackgroundTask {
	oneHourAgo := time.Now().Add(-time.Hour).Unix()

	var task *model.BackgroundTask
	err := gdb.Where("status = ?", model.TaskStatusFailed).
		Where("updated_at >= ?", oneHourAgo).
		Where("retry_count < max_retry").
		First(&task).Error

	if err != nil {
		return nil
	}

	// 更新任务状态为 pending，增加重试次数
	err = Transaction(func(tx *gorm.DB) error {
		return tx.Model(task).Updates(map[string]interface{}{
			"status":      model.TaskStatusPending,
			"retry_count": task.RetryCount + 1,
			"progress":    0,
			"updated_at":  time.Now().Unix(),
		}).Error
	})

	if err != nil {
		return nil
	}

	return task
}

func UpdateTaskStatus(taskID model.ID, progress int, status, message string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"message":    message,
		"updated_at": now.Unix(),
	}
	if !(progress == 0 && status == model.TaskStatusFailed) {
		updates["progress"] = progress
	}
	if progress == 10 && status == model.TaskStatusRunning {
		updates["started_at"] = now.Unix()
	}
	if status != "" {
		updates["status"] = status
	}
	if status == model.TaskStatusSuccess || status == model.TaskStatusFailed {
		updates["finished_at"] = now.Unix()
	}

	return Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.BackgroundTask{}).Where("id = ?", taskID).Updates(updates).Error
	})
}

func GetRunningTasks() ([]*model.BackgroundTask, error) {
	var tasks []*model.BackgroundTask
	err := gdb.Where("status = ?", model.TaskStatusRunning).Find(&tasks).Error
	return tasks, err
}

func GetOneWaitingTask() *model.BackgroundTask {
	var task model.BackgroundTask
	gdb.Where("status = ?", model.TaskStatusWaiting).
		First(&task)

	if task.ID > 0 {
		return &task
	}
	return nil
}

func ResetRunningTasksToWaiting() error {
	return Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.BackgroundTask{}).
			Where("status = ?", model.TaskStatusRunning).
			Update("status", model.TaskStatusWaiting).Error
	})
}

// GetLastSuccessfulSyncTrendTask 获取上次成功完成的同步趋势任务
// 返回任务的完成时间，如果没有找到则返回 0
func GetLastSuccessfulSuccessfulTrendTask(taskType string) int64 {
	var task model.BackgroundTask
	err := gdb.Where("task_type = ? AND status = ?", taskType, model.TaskStatusSuccess).
		Order("finished_at DESC").
		First(&task).Error

	if err != nil {
		return 0
	}
	return task.FinishedAt
}

// HasTrendTypeSyncedInDays 检查指定趋势类型在指定天数内是否同步成功
// 通过检查 message 字段中是否包含趋势类型标识来判断
func HasTrendTypeSyncedInDays(taskType string, trendType string, days int) bool {
	threshold := time.Now().AddDate(0, 0, -days).Unix()

	var count int64
	err := gdb.Model(&model.BackgroundTask{}).
		Where("task_type = ? AND status = ? AND finished_at >= ?", taskType, model.TaskStatusSuccess, threshold).
		Where("message LIKE ?", "%"+trendType+"%").
		Count(&count).Error

	if err != nil {
		return false
	}
	return count > 0
}
