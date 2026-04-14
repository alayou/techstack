package model

import "github.com/spf13/afero"

const (
	TaskStatusWaiting  = "waiting"  // 未开始，等待中，需要触发条件或手动开始
	TaskStatusPending  = "pending"  // 开始了，已经加入队列，等待程序执行
	TaskStatusRunning  = "running"  // 正在进行
	TaskStatusSuccess  = "success"  // 完成
	TaskStatusFailed   = "failed"   // 失败， 执行失败或者超时失败
	TaskStatusCanceled = "canceled" // 手动取消了
)

// BackgroundTask
// 后台异步任务
// 用于处理仓库解析、AI 分析、脚手架生成等长耗时操作
type BackgroundTask struct {
	ID         ID     `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"` // 主键ID
	UserID     int64  `json:"user_id,string,omitempty" gorm:"not null;index"`            // 所属用户ID
	TaskType   string `gorm:"size:50;not null;index" json:"task_type"`                   // 任务类型：parse_repo/gen_techstack/gen_scaffold
	PubRepoID  int64  `json:"pub_repo_id,string,omitempty" gorm:"index"`                 // 关联公共仓库ID
	Status     string `gorm:"size:30;default:pending;index" json:"status"`               // 任务状态
	Progress   int    `gorm:"default:0" json:"progress"`                                 // 进度 0~100
	Message    string `gorm:"type:text" json:"message"`                                  // 日志或错误信息
	RetryCount int    `gorm:"default:0" json:"retry_count"`                              // 已重试次数
	MaxRetry   int    `gorm:"default:3" json:"max_retry"`                                // 最大重试次数
	StartedAt  int64  `gorm:"started_at" json:"started_at"`                              // 开始执行时间
	FinishedAt int64  `gorm:"finished_at" json:"finished_at"`                            // 执行结束时间
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime;not null"`                 // 创建时间
	UpdatedAt  int64  `json:"updated_at" gorm:"autoUpdateTime;not null"`                 // 更新时间

	RepoFs afero.Fs          `gorm:"-" json:"-"` // git仓库源码文件系统
	Extras map[string]string `gorm:"-" json:"-"`
}

func (BackgroundTask) TableName() string {
	return BackgroundTaskTableName
}

type LLMModelConfig struct {
	ApiKey   string `json:"apiKey" yaml:"apiKey"`
	BaseUrl  string `json:"baseUrl" yaml:"baseUrl"`
	Provider string `json:"provider" yaml:"provider"`
	Model    string `json:"model" yaml:"model"`
}
