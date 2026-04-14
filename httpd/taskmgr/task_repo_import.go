package taskmgr

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alayou/techstack/httpd/dao"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/pkg/pkgclient"
	"github.com/alayou/techstack/pkg/repofs"
	"github.com/google/osv-scalibr/extractor"
	"github.com/google/osv-scalibr/purl"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// failTaskAndRepo 在一个事务中同时更新任务状态和仓库状态为失败
func failTaskAndRepo(task *model.BackgroundTask, progress int, message string) {
	taskID := task.ID
	repoID := task.PubRepoID
	ty := task.TaskType
	dao.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// 更新任务状态
		taskUpdates := map[string]interface{}{
			"progress":    progress,
			"status":      model.TaskStatusFailed,
			"message":     message,
			"updated_at":  now.Unix(),
			"finished_at": now.Unix(),
		}
		if err := tx.Model(&model.BackgroundTask{}).Where("id = ?", taskID).Updates(taskUpdates).Error; err != nil {
			return err
		}
		// 更新仓库状态
		repoUpdates := map[string]interface{}{}
		if ty == TaskTypeAnalysisPublicRepo {
			repoUpdates["analysis_stutas"] = model.RepoStatusFailed
			repoUpdates["last_analyzed_at"] = now.Unix()
		}
		if ty == TaskTypeRepoImport {
			repoUpdates["import_status"] = model.RepoStatusFailed
		}
		if len(repoUpdates) == 0 {
			return nil
		}
		return tx.Model(&model.PublicRepo{}).Where("id = ?", repoID).Updates(repoUpdates).Error
	})
}

// executePublicRepoScanTask 执行公共仓库扫描、解析、模块&依赖生成
func executeRepoImportTask(task *model.BackgroundTask) (err error) {
	err = dao.UpdateTaskStatus(task.ID, 10, model.TaskStatusRunning, "开始拉取仓库信息...")
	if err != nil {
		return fmt.Errorf("开始拉取仓库信息: %v", err)
	}
	repoID := task.PubRepoID
	if repoID == 0 {
		return fmt.Errorf("公共仓库ID不存在")
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
	repoURL = publicRepo.RepoURL
	branch = publicRepo.DefaultBranch
	// 1. 使用go-git库直接获取仓库引用快照
	err = dao.UpdateTaskStatus(task.ID, 20, model.TaskStatusRunning, fmt.Sprintf("正在下载仓库代码: %s", repoURL))
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	gitFs, commitHash, err := repofs.GitCloneFromRemoteToFs(context.Background(), repoURL, branch)
	if err != nil {
		failTaskAndRepo(task, 20, fmt.Sprintf("下载仓库代码失败: %v", err))
		return fmt.Errorf("下载仓库代码失败: %v", err)
	}

	// 2. 使用 SCALIBR 解析项目语言、依赖文件
	err = dao.UpdateTaskStatus(task.ID, 40, model.TaskStatusRunning, "正在解析模块与依赖...")
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	scanRes := repofs.ScanFromFs(context.Background(), afero.NewIOFS(gitFs))
	ipkgs := repofs.VersionFormat(scanRes.Inventory.Packages)
	err = dao.UpdateTaskStatus(task.ID, 40, model.TaskStatusRunning, "正在解析模块与依赖...")
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	// 3. 生成 RepoDependency 记录
	err = dao.UpdateTaskStatus(task.ID, 50, model.TaskStatusRunning, "保存依赖数据...")
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	var deps []*model.RepoDependency
	deps, err = CreateRepoModulesFromPackages(repoID, ipkgs)
	if err != nil {
		failTaskAndRepo(task, 50, fmt.Sprintf("保存依赖数据失败: %v", err))
		return fmt.Errorf("保存依赖数据失败: %v", err)
	}

	// ===========================================================================================
	// 根据仓库依赖包更新全局包数据
	err = dao.UpdateTaskStatus(task.ID, 52, model.TaskStatusRunning, "更新包数据...")
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	err = UpserPackages(context.TODO(), deps)
	if err != nil {
		failTaskAndRepo(task, 52, fmt.Sprintf("更新包数据数据失败: %v", err))
		return fmt.Errorf("更新包数据失败: %v", err)
	}
	// ===========================================================================================

	// 新建仓库依赖与包索引关联
	err = dao.UpdateTaskStatus(task.ID, 56, model.TaskStatusRunning, "更新包索引...")
	if err != nil {
		return fmt.Errorf("更新包索引: %v", err)
	}
	err = IndexRepoDependencyPkg(repoID, deps)
	if err != nil {
		failTaskAndRepo(task, 56, fmt.Sprintf("更新包索引数据失败: %v", err))
		return fmt.Errorf("更新包索引数据失败: %v", err)
	}

	err = dao.Repo.UpdateRepoImportStatus(repoID, model.RepoStatusSuccess)
	if err != nil {
		return fmt.Errorf("更新仓库状态失败: %v", err)
	}
	err = dao.Repo.UpdateRepoImportStatus(repoID, model.RepoStatusSuccess)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}
	//  自动创建仓库分析任务
	go CreateRepoAnalysisTask(task.UserID, TaskTypeAnalysisPublicRepo, task.PubRepoID, commitHash, gitFs) // nolint
	return nil
}

// batchSize 每批保存的依赖数量，避免 SQL 语句过长
const batchSize = 1000

func CreateRepoModulesFromPackages(repoID int64, pkgs []*extractor.Package) (deps []*model.RepoDependency, err error) {
	deps = make([]*model.RepoDependency, 0, len(pkgs))
	var repeType = "public"
	// 用于去重：key 为 PURL，相同 PURL 只保留第一条记录
	seenPURLs := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		purlType := ""
		if pkg.PURL() != nil {
			purlType = pkg.PURL().Type
		}
		if purlType == "" {
			continue
		}
		purlStr := pkg.PURL().String()
		// 跳过重复的 PURL（idx_pub_dep_purl 唯一键保护）
		if seenPURLs[purlStr] {
			continue
		}
		seenPURLs[purlStr] = true

		sourceFile := ""
		if len(pkg.Locations) > 0 {
			sourceFile = pkg.Locations[0]
		}
		deps = append(deps, &model.RepoDependency{
			ID:         model.NewID(),
			RepoType:   repeType,
			RepoID:     repoID,
			Version:    pkg.Version,
			SourceFile: sourceFile,
			PURL:       purlStr,
		})
	}

	err = dao.Transaction(func(tx *gorm.DB) error {
		err := tx.Delete(new(model.RepoDependency), "repo_id=? and repo_type=?", repoID, repeType).Error
		if err != nil {
			return err
		}

		// 分批保存依赖，避免 SQL 语句过长；使用 ON CONFLICT DO NOTHING 跳过重复数据
		for i := 0; i < len(deps); i += batchSize {
			end := i + batchSize
			if end > len(deps) {
				end = len(deps)
			}
			batch := deps[i:end]
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(batch, batchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return
}

// repoDepPurlInfo 仓库依赖解析后的PURL信息
type repoDepPurlInfo struct {
	dep     *model.RepoDependency
	system  string
	name    string
	version string
}

// UpserPackages 批量解析PURL对应的包信息
// 根据仓库依赖，解析purl，判断packageversion和packages数据库是否存在，不存在则插入，存在则跳过
// 支持批量更新和插入
func UpserPackages(ctx context.Context, deps []*model.RepoDependency) error {
	if len(deps) == 0 {
		return nil
	}

	// 用于去重：key 为 PURL，相同 PURL 只保留第一条记录
	seenPURLs := make(map[string]bool)
	uniqueDeps := make([]*model.RepoDependency, 0, len(deps))
	for _, dep := range deps {
		if dep == nil || dep.PURL == "" {
			continue
		}
		if seenPURLs[dep.PURL] {
			continue
		}
		seenPURLs[dep.PURL] = true
		uniqueDeps = append(uniqueDeps, dep)
	}

	// 收集所有 PURL 和包信息
	pkgInfos := make([]repoDepPurlInfo, 0, len(uniqueDeps))
	for _, dep := range uniqueDeps {
		p, err := purl.FromString(dep.PURL)
		if err != nil {
			continue
		}
		system, name, version := pkgclient.PurlToPkgInfo(p)
		if system == "" || name == "" {
			continue
		}
		pkgInfos = append(pkgInfos, repoDepPurlInfo{
			dep:     dep,
			system:  system,
			name:    name,
			version: version,
		})
	}

	// 批量获取包信息
	batchSize := 100
	for i := 0; i < len(pkgInfos); i += batchSize {
		end := i + batchSize
		if end > len(pkgInfos) {
			end = len(pkgInfos)
		}
		batch := pkgInfos[i:end]

		// 收集需要查询的包信息
		var pkgList []*pkgclient.PackageInfo
		var pkgInfoIndices []int

		for j, pi := range batch {
			pkgInfo, err := pkgclient.GetPackageInfoByPurl(ctx, pi.dep.PURL)
			if err != nil {
				continue
			}
			if pkgInfo.Name == "" {
				log.Error().Msg("意外解析道库名称为空")
				continue
			}
			pkgInfo.PURLType = pi.system
			pkgList = append(pkgList, pkgInfo)
			pkgInfoIndices = append(pkgInfoIndices, j)
		}

		// 处理每个包：先检查/插入 Package，再检查/插入 PackageVersion
		if err := upsertPackagesAndVersions(ctx, pkgList, batch, pkgInfoIndices); err != nil {
			return err
		}
	}

	return nil
}

// upsertPackagesAndVersions 批量插入或更新 Package 和 PackageVersion 记录
func upsertPackagesAndVersions(ctx context.Context, pkgList []*pkgclient.PackageInfo, deps []repoDepPurlInfo, indices []int) error {
	if len(pkgList) == 0 {
		return nil
	}

	return dao.Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()

		// 第一步：处理 Package（通过 purl_type + name 判断是否存在）
		packageMap := make(map[string]*model.Package) // key: purl_type:name

		for i, pkg := range pkgList {
			if pkg == nil {
				continue
			}
			key := pkg.PURLType + ":" + pkg.Name
			packageMap[key] = &model.Package{
				ID:             model.NewID(),
				Name:           pkg.Name,
				PurlType:       pkg.PURLType,
				Description:    pkg.Description,
				NormalizedName: strings.ToLower(pkg.Name),
				HomepageURL:    pkg.HomepageURL,
				RepositoryURL:  pkg.RepoURL,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			_ = i // 未使用的变量，保留索引
		}

		// 查询已存在的 Package
		for key, pkg := range packageMap {
			var existing model.Package
			err := tx.Where("purl_type = ? AND name = ?", pkg.PurlType, pkg.Name).First(&existing).Error
			if err == nil {
				// Package 已存在，使用已存在的 ID
				packageMap[key] = &existing
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// Package 不存在，插入新记录
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(pkg).Error; err != nil {
					return err
				}
				// 重新查询获取 ID
				tx.Where("purl_type = ? AND name = ?", pkg.PurlType, pkg.Name).First(pkg)
			} else {
				return err
			}
		}

		// 第二步：处理 PackageVersion（通过 PURL 判断是否存在）
		versions := make([]*model.PackageVersion, 0, len(pkgList))
		for i, idx := range indices {
			if idx >= len(deps) {
				continue
			}
			dep := deps[idx].dep
			pkg := pkgList[i]
			if pkg == nil {
				continue
			}

			key := pkg.PURLType + ":" + pkg.Name
			packageRecord, ok := packageMap[key]
			if !ok {
				continue
			}

			// 检查 PackageVersion 是否已存在（通过 PURL）
			var existingVer model.PackageVersion
			err := tx.Where("purl = ?", dep.PURL).First(&existingVer).Error
			if err == nil {
				// 已存在，跳过
				continue
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				// 不存在，插入新记录
				ver := &model.PackageVersion{
					ID:             model.NewID(),
					PackageID:      packageRecord.ID.Int64(),
					Version:        dep.Version,
					PURL:           dep.PURL,
					PublishedAt:    pkg.ReleasedAt.Unix(),
					PublishedAtStr: pkg.ReleasedAt.Format("2006-01-02"),
				}
				versions = append(versions, ver)
			} else {
				return err
			}
		}

		// 批量插入 PackageVersion
		if len(versions) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(versions, batchSize).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// UpserPackage 单条解析PURL对应的包信息
func UpserPackage(ctx context.Context, dep *model.RepoDependency) error {
	if dep == nil || dep.PURL == "" {
		return nil
	}

	p, err := purl.FromString(dep.PURL)
	if err != nil {
		return err
	}
	system, name, _ := pkgclient.PurlToPkgInfo(p)
	if system == "" || name == "" {
		return fmt.Errorf("invalid purl: %s", dep.PURL)
	}

	pkgInfo, err := pkgclient.GetPackageInfoByPurl(ctx, dep.PURL)
	if err != nil {
		return err
	}
	pkgInfo.PURLType = system

	now := time.Now().Unix()

	return dao.Transaction(func(tx *gorm.DB) error {
		// 处理 Package
		var pkg model.Package
		err := tx.Where("purl_type = ? AND name = ?", system, name).First(&pkg).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Package 不存在，插入新记录
			pkg = model.Package{
				ID:             model.NewID(),
				Name:           pkgInfo.Name,
				PurlType:       pkgInfo.PURLType,
				Description:    pkgInfo.Description,
				NormalizedName: strings.ToLower(pkgInfo.Name),
				HomepageURL:    pkgInfo.HomepageURL,
				RepositoryURL:  pkgInfo.RepoURL,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&pkg).Error; err != nil {
				return err
			}
			// 重新查询获取完整记录
			tx.Where("purl_type = ? AND name = ?", system, name).First(&pkg)
		} else if err != nil {
			return err
		}

		// 处理 PackageVersion
		var existingVer model.PackageVersion
		err = tx.Where("purl = ?", dep.PURL).First(&existingVer).Error
		if err == nil {
			// 已存在，跳过
			return nil
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// 不存在，插入新记录
			ver := &model.PackageVersion{
				ID:             model.NewID(),
				PackageID:      pkg.ID.Int64(),
				Version:        dep.Version,
				PURL:           dep.PURL,
				PublishedAt:    pkgInfo.ReleasedAt.Unix(),
				PublishedAtStr: pkgInfo.ReleasedAt.Format("2006-01-02"),
			}
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(ver).Error
		}
		return err
	})
}

// IndexRepoDependencyPkg 新建包与仓库依赖索引关联
// 将 RepoDependency 与 Package 和 PackageVersion 关联，如果没有则跳过
func IndexRepoDependencyPkg(repoID int64, deps []*model.RepoDependency) error {
	if len(deps) == 0 {
		return nil
	}

	// 收集索引记录
	var pkgIndexes []*model.RepoPkgIndex            // RepoDependency 与 Package 的关联
	var versionIndexes []*model.RepoPkgVersionIndex // RepoDependency 与 PackageVersion 的关联

	for _, dep := range deps {
		if dep == nil || dep.PURL == "" {
			continue
		}

		// 通过 PURL 查询 PackageVersion
		var pkgVersion model.PackageVersion
		err := dao.View(func(tx *gorm.DB) error {
			return tx.Where("purl = ?", dep.PURL).First(&pkgVersion).Error
		})
		if err != nil {
			// PackageVersion 不存在，跳过
			continue
		}

		// 创建 RepoDependency 与 PackageVersion 的关联
		versionIndexes = append(versionIndexes, &model.RepoPkgVersionIndex{
			RepoID: repoID,
			DepID:  dep.ID.Int64(),
			PkgID:  pkgVersion.ID.Int64(),
		})

		// 创建 RepoDependency 与 Package 的关联
		pkgIndexes = append(pkgIndexes, &model.RepoPkgIndex{
			RepoID: repoID,
			DepID:  dep.ID.Int64(),
			PkgID:  pkgVersion.PackageID,
		})
	}

	if len(pkgIndexes) == 0 && len(versionIndexes) == 0 {
		return nil
	}

	return dao.Transaction(func(tx *gorm.DB) error {
		// 删除旧的索引关联
		if err := tx.Delete(new(model.RepoPkgIndex), "repo_id=?", repoID).Error; err != nil {
			return err
		}
		if err := tx.Delete(new(model.RepoPkgVersionIndex), "repo_id=?", repoID).Error; err != nil {
			return err
		}

		// 批量插入新的索引关联
		if len(pkgIndexes) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(pkgIndexes, batchSize).Error; err != nil {
				return err
			}
		}
		if len(versionIndexes) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(versionIndexes, batchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
