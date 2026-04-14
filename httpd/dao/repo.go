package dao

import (
	"strings"
	"time"

	"github.com/alayou/techstack/model"
	"gorm.io/gorm"
)

type repo struct {
}

var Repo = new(repo)

func (*repo) UpdateRepoImportStatus(repoID int64, status string) error {
	return Transaction(func(tx *gorm.DB) error {
		values := map[string]interface{}{
			"import_status":    status,
			"last_analyzed_at": time.Now().Unix(),
		}
		return tx.Model(&model.PublicRepo{}).
			Where("id = ?", repoID).
			Updates(values).Error
	})
}

func (*repo) UpdateRepoAnalysisStatus(repoID int64, status string) error {
	return Transaction(func(tx *gorm.DB) error {
		values := map[string]interface{}{
			"analysis_status":  status,
			"last_analyzed_at": time.Now().Unix(),
		}
		return tx.Model(&model.PublicRepo{}).
			Where("id = ?", repoID).
			Updates(values).Error
	})
}

func (*repo) UpdateRepoAnalysisStatusAndSummary(repoID int64, summary string) error {
	return Transaction(func(tx *gorm.DB) error {
		values := map[string]interface{}{
			"analysis_status":  model.RepoStatusSuccess,
			"analysis_summary": summary,
			"last_analyzed_at": time.Now().Unix(),
		}
		return tx.Model(&model.PublicRepo{}).
			Where("id = ?", repoID).
			Updates(values).Error
	})
}

func (*repo) Create(publicRepo *model.PublicRepo) error {
	publicRepo.RepoURL = strings.TrimSuffix(publicRepo.RepoURL, ".git")
	return gdb.Create(&publicRepo).Error
}
