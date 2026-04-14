package taskmgr

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/alayou/techstack/httpd/dao"
	"github.com/alayou/techstack/model"
	"github.com/spf13/afero"
	"gorm.io/gorm"
)

const (
	TaskTypeRepoImport         = "repo_import"          // 导入项目
	TaskTypeAnalysisPublicRepo = "analysis_public_repo" // 分析公共仓库
)

type TaskExecutor func(task *model.BackgroundTask) error

// 任务执行器注册表（只读，无并发风险）
var taskExecutors = make(map[string]TaskExecutor)

// 任务队列（Worker Pool）
var taskChan = make(chan *model.BackgroundTask, 10)

const workerCount = 5

func RegisterTaskExecutor(taskType string, executor TaskExecutor) {
	taskExecutors[taskType] = executor
}

// StartWorkerPool 启动协程池（服务启动时调用）
func StartWorkerPool(ctx context.Context) {
	// 重置所有运行中的任务状态为 waiting
	_ = dao.ResetRunningTasksToWaiting()

	// 先启动 worker
	for i := 0; i < workerCount; i++ {
		go worker(ctx)
	}

}

// tryEnqueueTask 尝试非阻塞地将任务加入队列
func tryEnqueueTask(task *model.BackgroundTask) bool {
	select {
	case taskChan <- task:
		return true
	default:
		return false
	}
}

func worker(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-taskChan:
			if !ok {
				return
			}
			executeTask(task)
		case <-time.After(time.Minute):
			// 先处理waiting的任务
			task := dao.GetOneWaitingTask()
			if task != nil {
				tryEnqueueTask(task)
			}
			// 先处理失败的任务
			task = dao.GetLastStopTask()
			if task != nil {
				tryEnqueueTask(task)
			}
		}
	}
}

// executeTask 执行任务（带panic防护）
func executeTask(task *model.BackgroundTask) {
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("任务崩溃: %v\n%s", r, debug.Stack())
			_ = dao.UpdateTaskStatus(task.ID, task.Progress, model.TaskStatusFailed, task.Message+"\n"+errMsg)
		}
	}()

	// 获取执行器
	executor, ok := taskExecutors[task.TaskType]
	if !ok {
		_ = dao.UpdateTaskStatus(task.ID, 0, model.TaskStatusFailed, "未注册的任务类型")
		return
	}

	// 只有当任务状态不是 running 时才更新（避免重复更新从 GetAndLockOnePendingTask 获取的任务）
	if task.Status != model.TaskStatusRunning {
		now := time.Now()
		_ = dao.Transaction(func(tx *gorm.DB) error {
			return tx.Model(task).Updates(map[string]interface{}{
				"status":     model.TaskStatusRunning,
				"started_at": now.Unix(),
			}).Error
		})
	}

	// 执行业务逻辑
	if err := executor(task); err != nil {
		_ = dao.UpdateTaskStatus(task.ID, 0, model.TaskStatusFailed, task.Message+"\n"+err.Error())
		return
	}

	// 执行成功
	_ = dao.UpdateTaskStatus(task.ID, 100, model.TaskStatusSuccess, "任务完成")
}

func CreateRepoAnalysisTask(userID int64, taskType string, repoID int64, commitHash string, repoFs afero.Fs) (*model.BackgroundTask, error) {
	task := &model.BackgroundTask{
		UserID:     userID,
		TaskType:   taskType,
		Status:     model.TaskStatusPending,
		Progress:   0,
		RetryCount: 0,
		MaxRetry:   3,
		RepoFs:     repoFs,
		PubRepoID:  repoID,
		Extras: map[string]string{
			"commitHash": commitHash,
		},
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
func CreateTask(userID int64, taskType string, repoID int64) (*model.BackgroundTask, error) {
	task := &model.BackgroundTask{
		UserID:     userID,
		TaskType:   taskType,
		Status:     model.TaskStatusPending,
		Progress:   0,
		RetryCount: 0,
		MaxRetry:   3,
	}

	task.PubRepoID = repoID
	// 入库
	if err := dao.Transaction(func(tx *gorm.DB) error {
		return tx.Create(task).Error
	}); err != nil {
		return nil, err
	}

	startTaskAsync(task)
	return task, nil
}

func StartTaskAsync(task *model.BackgroundTask) {
	startTaskAsync(task)
}
func startTaskAsync(task *model.BackgroundTask) {
	select {
	case taskChan <- task:
	default:
		_ = dao.UpdateTaskStatus(task.ID, 0, model.TaskStatusFailed, "任务队列已满，请稍后重试")
	}
}

func init() {
	RegisterTaskExecutor(TaskTypeRepoImport, executeRepoImportTask)
	RegisterTaskExecutor(TaskTypeAnalysisPublicRepo, executeAnalysisPublicRepoTask)
}
