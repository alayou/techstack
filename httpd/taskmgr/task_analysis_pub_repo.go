package taskmgr

import (
	"context"
	"fmt"

	"github.com/alayou/techstack/global"
	"github.com/alayou/techstack/httpd/dao"
	"github.com/alayou/techstack/httpd/llmutil"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/pkg/repofs"
	"gorm.io/gorm"
)

// executeAnalysisPublicRepoTask 执行分析开源的仓库
// 通过调用openai模型分析项目,分析源码解析,获取代码风格、项目架构
func executeAnalysisPublicRepoTask(task *model.BackgroundTask) (err error) {
	err = dao.UpdateTaskStatus(task.ID, 10, model.TaskStatusRunning, "开始拉取仓库信息...")
	if err != nil {
		return fmt.Errorf("开始拉取仓库信息失败: %v", err)
	}
	repoID := task.PubRepoID
	if repoID == 0 {
		return fmt.Errorf("公共仓库ID不存在")
	}
	if task.Extras == nil {
		task.Extras = make(map[string]string)
	}
	var publicRepo model.PublicRepo
	var repoURL string
	var branch string
	err = dao.View(func(tx *gorm.DB) error {
		return tx.First(&publicRepo, repoID).Error
	})
	if err != nil {
		return fmt.Errorf("查询公共仓库失败: %v", err)
	}
	var analysis model.RepoTechAnalysis
	repoURL = publicRepo.RepoURL
	branch = publicRepo.DefaultBranch
	// 使用go-git库直接获取仓库引用快照
	err = dao.UpdateTaskStatus(task.ID, 20, model.TaskStatusRunning, fmt.Sprintf("正在下载仓库代码: %s", repoURL))
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	if task.RepoFs == nil {
		repoFs, commitHash, err := repofs.GitCloneFromRemoteToFs(context.Background(), repoURL, branch)
		if err != nil {
			failTaskAndRepo(task, 20, fmt.Sprintf("下载仓库代码失败: %v", err))
			return fmt.Errorf("下载仓库代码失败: %v", err)
		}
		task.RepoFs = repoFs
		task.Extras["commitHash"] = commitHash
	}

	err = dao.UpdateTaskStatus(task.ID, 30, model.TaskStatusRunning, fmt.Sprintf("初始化分析表数据: %s", repoURL))
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	err = dao.Transaction(func(tx *gorm.DB) error {
		return tx.Model(new(model.RepoTechAnalysis)).Where("repo_id=?", repoID).
			Assign(model.RepoTechAnalysis{
				RepoType:   "public",
				RepoID:     repoID,
				Branch:     branch,
				CommitHash: task.Extras["commitHash"],
			}).FirstOrCreate(&analysis).Error
	})
	if err != nil {
		failTaskAndRepo(task, 30, fmt.Sprintf("初始化分析表数据失败: %v", err))
		return fmt.Errorf("初始化分析表数据失败: %v", err)
	}
	err = dao.UpdateTaskStatus(task.ID, 30, model.TaskStatusRunning, fmt.Sprintf("初始化分析表数据: %s", repoURL))
	if err != nil {
		return fmt.Errorf("Agent正在进行分析你的仓库: %v", err)
	}
	repoFsTools := repofs.NewLLMFsTools(task.RepoFs)
	repoFsTools = append(repoFsTools, dao.NewRepoTools(task.PubRepoID)...)
	var summary string
	summary, err = llmutil.AgentAnalysizeRepo(context.Background(), &global.Config.LLM, publicRepo.FormtString(), repoFsTools)
	if err != nil {
		failTaskAndRepo(task, 30, fmt.Sprintf("Agent正在进行分析你的仓库: %v", err))
		return fmt.Errorf("Agent正在进行分析你的仓库: %v", err)
	}
	err = dao.Repo.UpdateRepoAnalysisStatusAndSummary(repoID, summary)
	if err != nil {
		return fmt.Errorf("更新总结信息失败: %v", err)
	}
	err = dao.UpdateTaskStatus(task.ID, 30, model.TaskStatusRunning, fmt.Sprintf("Agent正在进行分析你的仓库: %s", repoURL))
	if err != nil {
		return fmt.Errorf("完成分析: %v", err)
	}

	return
}
