package tools

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"github.com/alayou/techstack/global"
	"github.com/alayou/techstack/httpd/dao"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/utils"
	"github.com/urfave/cli/v3"
)

const (
	// DuckDB 文件默认路径
	DefaultDuckDBFile = "go_index.duckdb"
	// 批处理大小
	importBatchSize = 1000
	// DuckDB 批处理大小
	duckdbBatchSize = 10000
)

// GoModuleEntry 从 DuckDB 读取的 Go 模块条目
type GoModuleEntry struct {
	Path      string    `json:"Path"`
	Version   string    `json:"Version"`
	Timestamp time.Time `json:"Timestamp"`
}

// ImportGolangModules 从 DuckDB 导入 Go 模块数据到 PostgreSQL
func ImportGolangModules(ctx context.Context, c *cli.Command) (err error) {
	// 获取可执行文件路径并加载配置文件
	exepath, _ := os.Executable()
	global.Config.ConfigFile = filepath.Join(filepath.Dir(exepath), "config.yml")
	_, err = utils.LoadYAMLConfig(global.Config.ConfigFile, &global.Config)
	if err != nil {
		log.Err(err).Str("ConfigFile", global.Config.ConfigFile).Msg("cannot load config file.")
		return err
	}

	// 获取 DuckDB 文件路径参数
	duckdbPath := c.Args().First()
	if duckdbPath == "" {
		// 默认使用当前目录下的 DuckDB 文件
		pwd, _ := os.Getwd()
		duckdbPath = filepath.Join(pwd, DefaultDuckDBFile)
	}

	// 检查文件是否存在
	if _, err := os.Stat(duckdbPath); os.IsNotExist(err) {
		log.Err(err).Str("File", duckdbPath).Msg("DuckDB file not found")
		return err
	}

	fmt.Printf("正在使用 DuckDB 导入 Go 模块数据: %s\n", duckdbPath)

	// 连接 DuckDB
	db, err := sql.Open("duckdb", duckdbPath)
	if err != nil {
		log.Err(err).Msg("failed to open DuckDB")
		return err
	}
	defer db.Close()

	// 验证连接
	if err := db.Ping(); err != nil {
		log.Err(err).Msg("failed to ping DuckDB")
		return err
	}
	fmt.Println("DuckDB 连接成功")

	// 检查 go_modules 表是否存在
	var tableExists bool
	if err := db.QueryRow("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'go_modules')").Scan(&tableExists); err != nil {
		log.Err(err).Msg("failed to check if go_modules table exists")
		return err
	}
	if !tableExists {
		return fmt.Errorf("go_modules 表不存在，请先运行 fetch-all-gomod2duck.py 同步数据")
	}

	// 获取总记录数
	var totalCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM go_modules").Scan(&totalCount); err != nil {
		log.Err(err).Msg("failed to count records")
		return err
	}
	fmt.Printf("DuckDB 中共有 %d 条模块版本记录\n", totalCount)

	// 获取唯一模块数
	var moduleCount int
	if err := db.QueryRow("SELECT COUNT(DISTINCT Path) FROM go_modules").Scan(&moduleCount); err != nil {
		log.Err(err).Msg("failed to count unique modules")
		return err
	}
	fmt.Printf("DuckDB 中共有 %d 个唯一 Go 模块\n", moduleCount)

	// 初始化目标数据库连接
	fmt.Println("正在连接目标数据库...")
	err = dao.Init(global.Config.Database.Driver, global.Config.Database.DSN)
	if err != nil {
		log.Err(err).Msg("failed to initialize database")
		return err
	}
	fmt.Println("目标数据库连接成功")

	// 获取已存在的 Go 库（按 package_name 去重）
	fmt.Println("检查已存在的 Go 库...")
	existingLibs := make(map[string]bool)
	var existing []struct {
		Name     string
		PurlType string
	}
	dao.View(func(tx *gorm.DB) error {
		return tx.Model(&model.Package{}).Select("name", "purl_type").Find(&existing).Error
	})
	for _, lib := range existing {
		key := lib.Name
		existingLibs[key] = true
	}
	fmt.Printf("目标数据库中已有 %d 条 Go 库记录\n", len(existingLibs))

	// 构建排除列表的 SQL 条件
	var excludeConditions []string
	for i := 0; i < len(existingLibs); i += duckdbBatchSize {
		end := i + duckdbBatchSize
		if end > len(existingLibs) {
			end = len(existingLibs)
		}
		batch := make([]string, 0, end-i)
		keys := make([]string, 0, len(existingLibs))
		for key := range existingLibs {
			keys = append(keys, key)
		}
		for j := i; j < end; j++ {
			batch = append(batch, fmt.Sprintf("'%s'", keys[j]))
		}
		excludeConditions = append(excludeConditions, strings.Join(batch, ","))
	}

	// 统计新增模块数
	var newModuleCount int
	if len(excludeConditions) > 0 {
		excludeSQL := strings.Join(excludeConditions, ",")
		countSQL := fmt.Sprintf(`
			SELECT COUNT(DISTINCT Path) FROM go_modules
			WHERE Path NOT IN (%s)
		`, excludeSQL)
		if err := db.QueryRow(countSQL).Scan(&newModuleCount); err != nil {
			log.Err(err).Str("sql", countSQL).Msg("failed to count new modules")
			return err
		}
	} else {
		newModuleCount = moduleCount
	}

	fmt.Printf("新增 %d 个模块需要导入\n", newModuleCount)

	if newModuleCount == 0 {
		fmt.Println("没有新数据需要导入")
		return nil
	}

	// 构建查询 SQL - 获取所有模块版本记录
	querySQL := `
		SELECT 
			Path,
			Version,
			Timestamp
		FROM go_modules
	`
	if len(excludeConditions) > 0 {
		excludeSQL := strings.Join(excludeConditions, ",")
		querySQL = fmt.Sprintf("%s WHERE Path NOT IN (%s)", querySQL, excludeSQL)
	}
	// 按 Path 和 Timestamp 排序，确保最新版本在前
	querySQL = fmt.Sprintf("%s ORDER BY Path, Timestamp DESC", querySQL)

	// 统计实际要处理的记录数
	var processCount int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) t", querySQL)
	if err := db.QueryRow(countSQL).Scan(&processCount); err != nil {
		log.Err(err).Msg("failed to count process records")
		return err
	}
	fmt.Printf("实际处理 %d 条模块版本记录\n", processCount)

	// 批量从 DuckDB 读取数据并导入到目标数据库
	fmt.Printf("正在批量导入数据到目标数据库...\n")
	totalImported := 0
	offset := 0

	for {
		batchSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", querySQL, importBatchSize, offset)
		rows, err := db.Query(batchSQL)
		if err != nil {
			log.Err(err).Msg("failed to query from DuckDB")
			return err
		}

		var batch []struct {
			Path      string
			Version   string
			Timestamp time.Time
		}
		for rows.Next() {
			var entry struct {
				Path      string
				Version   string
				Timestamp time.Time
			}
			if err := rows.Scan(&entry.Path, &entry.Version, &entry.Timestamp); err != nil {
				rows.Close()
				log.Err(err).Msg("failed to scan row")
				return err
			}
			batch = append(batch, entry)
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		// 批量插入到目标数据库
		if err := dao.Transaction(func(tx *gorm.DB) error {
			// 用于跟踪当前批次中已处理的模块，避免重复创建同一模块的库记录
			processedModules := make(map[string]bool)

			for i := range batch {
				path := batch[i].Path
				version := batch[i].Version
				timestamp := batch[i].Timestamp

				// 清理无效的UTF-8字符
				if !utf8.ValidString(path) {
					path = strings.ToValidUTF8(path, "")
				}

				// 检查是否已处理过该模块
				isFirstVersionForModule := !processedModules[path]

				// 查找或创建 libraries 表记录
				var pkgmod model.Package
				result := tx.Where("name = ? AND purl_type = ?", path, "go").First(&pkgmod)
				if result.Error == gorm.ErrRecordNotFound {
					// 记录不存在，创建新的
					pkgmod = model.Package{
						Name:           path,
						PurlType:       "go",
						NormalizedName: strings.ToLower(path),
						Description:    "",
						CreatedAt:      time.Now().Unix(),
						UpdatedAt:      time.Now().Unix(),
					}
					if err := tx.Create(&pkgmod).Error; err != nil {
						log.Err(err).Msgf("failed to insert package: %s", path)
						return err
					}
					// 设置最新版本
					tx.Model(&pkgmod).Update("latest_version", version)
					processedModules[path] = true
				} else if result.Error != nil {
					log.Err(result.Error).Msgf("failed to query package: %s", path)
					return result.Error
				} else if isFirstVersionForModule {
					// 模块已存在但这是该批次第一次处理它，更新最新版本
					tx.Model(&pkgmod).Update("latest_version", version)
					processedModules[path] = true
				}

				// 创建 package_versions 表记录
				pkgmodVersion := model.PackageVersion{
					PackageID:      pkgmod.ID.Int64(),
					Version:        version,
					PURL:           fmt.Sprintf("pkg:golang/%s@%s", path, version),
					PublishedAtStr: timestamp.Format(time.RFC3339),
					PublishedAt:    timestamp.Unix(),
				}

				// 检查版本是否已存在
				var existingVer int64
				tx.Model(&model.PackageVersion{}).Where("package_id = ? AND version = ?", pkgmod.ID.Int64(), version).Count(&existingVer)
				if existingVer == 0 {
					if err := tx.Create(&pkgmodVersion).Error; err != nil {
						log.Err(err).Msgf("failed to insert package version: %s@%s", path, version)
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}

		totalImported += len(batch)
		offset += importBatchSize
		fmt.Printf("已导入: %d / %d\n", totalImported, processCount)

		if len(batch) < importBatchSize {
			break
		}
	}

	fmt.Printf("导入完成！共导入 %d 条数据\n", totalImported)
	return nil
}
