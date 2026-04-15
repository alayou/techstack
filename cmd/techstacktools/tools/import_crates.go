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
	"github.com/alayou/techstack/pkg/logger"
	"github.com/alayou/techstack/utils"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/urfave/cli/v3"
)

// crates.csv 字段顺序（官方固定）
// 0: id
// 1: name
// 2: updated_at
// 3: created_at
// 4: downloads
// 5: description
// 6: homepage
// 7: repository
// 8: documentation
// ...

// ImportSourceType 数据源类型
type ImportSourceType string

const (
	SourceDuckDB ImportSourceType = "duckdb"
	SourceCSV    ImportSourceType = "csv"
)

func ImportCrates(ctx context.Context, c *cli.Command) (err error) {
	// 获取可执行文件路径并加载配置文件
	exepath, _ := os.Executable()
	configPath := c.String("config")
	if configPath == "" {
		configPath = "config.yml"
	}
	if filepath.IsAbs(configPath) {
		global.Config.ConfigFile = filepath.Clean(configPath)
	} else {
		global.Config.ConfigFile = filepath.Join(filepath.Dir(exepath), configPath)
	}
	_, err = utils.LoadYAMLConfig(global.Config.ConfigFile, &global.Config)
	if err != nil {
		log.Err(err).Str("ConfigFile", global.Config.ConfigFile).Msg("cannot load config file.")
		return err
	}
	logger.InitLogger("debug", "")

	// 解析命令行参数
	sourceType := ImportSourceType(c.String("source"))
	if sourceType == "" {
		sourceType = SourceCSV // 默认 CSV 模式
	}

	var dataPath string
	switch sourceType {
	case SourceDuckDB:
		// DuckDB 模式：第二个参数是 duckdb 文件路径
		dataPath = c.Args().First()
		if dataPath == "" {
			// 默认使用当前目录下的 crates_io.duckdb
			dataPath = "crates_io.duckdb"
		}
	case SourceCSV:
		// CSV 模式：第二个参数是 csv 文件路径
		dataPath = c.Args().First()
		if dataPath == "" {
			fmt.Println("请提供 CSV 文件路径")
			fmt.Println("用法: techstacktools import crates --source csv <path/to/crates.csv>")
			fmt.Println("或:   techstacktools import crates --source duckdb [path/to/crates_io.duckdb]")
			return fmt.Errorf("缺少数据文件路径参数")
		}
	default:
		return fmt.Errorf("不支持的数据源类型: %s，支持的类型: duckdb, csv", sourceType)
	}

	fmt.Printf("正在使用 %s 模式导入数据: %s\n", sourceType, dataPath)

	// 初始化目标数据库连接
	fmt.Println("正在连接目标数据库...")
	err = dao.Init(global.Config.Database.Driver, global.Config.Database.DSN)
	if err != nil {
		log.Err(err).Msg("failed to initialize database")
		return err
	}
	fmt.Println("目标数据库连接成功")

	// 获取已存在的库（name + purl_type 组合）
	fmt.Println("检查已存在的库...")
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
	fmt.Printf("目标数据库中已有 %d 条库记录\n", len(existingLibs))

	// 根据数据源类型执行不同的导入逻辑
	switch sourceType {
	case SourceDuckDB:
		err = importFromDuckDB(dataPath, existingLibs)
	case SourceCSV:
		err = importFromCSV(dataPath, existingLibs)
	}

	return err
}

// importFromDuckDB 从 DuckDB 数据库导入 crates 数据
func importFromDuckDB(duckdbPath string, existingLibs map[string]bool) error {
	// 检查文件是否存在
	if _, err := os.Stat(duckdbPath); os.IsNotExist(err) {
		log.Err(err).Str("File", duckdbPath).Msg("DuckDB file not found")
		return err
	}

	fmt.Println("正在连接 DuckDB 数据库...")

	// 直接打开 DuckDB 文件
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

	// 检查 crates 表是否存在
	var tableExists int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'crates'").Scan(&tableExists); err != nil {
		log.Err(err).Msg("failed to check crates table")
		return err
	}
	if tableExists == 0 {
		return fmt.Errorf("crates 表不存在于 DuckDB 文件: %s", duckdbPath)
	}

	// 获取总记录数
	var totalCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM crates").Scan(&totalCount); err != nil {
		log.Err(err).Msg("failed to count records")
		return err
	}
	fmt.Printf("DuckDB crates 表共 %d 条记录\n", totalCount)

	// 构建排除条件
	batchSize := 10000
	var excludeConditions []string
	keys := make([]string, 0, len(existingLibs))
	for key := range existingLibs {
		keys = append(keys, key)
	}

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := make([]string, 0, end-i)
		for j := i; j < end; j++ {
			batch = append(batch, fmt.Sprintf("'%s'", keys[j]))
		}
		excludeConditions = append(excludeConditions, strings.Join(batch, ","))
	}

	// 计算新增记录数
	var newRecordsCount int
	if len(excludeConditions) > 0 {
		excludeSQL := strings.Join(excludeConditions, ",")
		countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM crates WHERE name NOT IN (%s)`, excludeSQL)
		if err := db.QueryRow(countSQL).Scan(&newRecordsCount); err != nil {
			log.Err(err).Msg("failed to count new records")
			return err
		}
	} else {
		newRecordsCount = totalCount
	}

	fmt.Printf("新增 %d 条记录需要导入\n", newRecordsCount)

	if newRecordsCount == 0 {
		fmt.Println("没有新数据需要导入")
		return nil
	}

	// 批量导入数据
	fmt.Printf("正在批量导入数据到目标数据库...\n")
	importBatchSize := 1000
	totalImported := 0
	offset := 0

	// 构建查询 SQL - DuckDB 的 crates 表字段
	// 注意：DuckDB 的 crates 表中 created_at/updated_at 是 TIMESTAMP WITH TIME ZONE 类型
	// 需要转换为 TIMESTAMP 再提取 epoch，并转换为 BIGINT
	querySQL := `
		SELECT 
			name,
			'cargo' as purl_type,
			CASE WHEN description IS NULL THEN '' ELSE description END as description,
			LOWER(name) as normalized_name,
			CASE WHEN homepage IS NULL THEN '' ELSE homepage END as homepage,
			CASE WHEN repository IS NULL THEN '' ELSE repository END as repository,
			CAST(EPOCH(CAST(created_at AS TIMESTAMP)) AS BIGINT) as created_at,
			CAST(EPOCH(CAST(updated_at AS TIMESTAMP)) AS BIGINT) as updated_at
		FROM crates
	`

	if len(excludeConditions) > 0 {
		excludeSQL := strings.Join(excludeConditions, ",")
		querySQL = fmt.Sprintf("%s WHERE name NOT IN (%s) and purl_type = 'cargo'", querySQL, excludeSQL)
	}

	// 按批次处理数据
	for {
		batchSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", querySQL, importBatchSize, offset)
		rows, err := db.Query(batchSQL)
		if err != nil {
			log.Err(err).Msg("failed to query from DuckDB")
			return err
		}

		var batch []model.Package
		for rows.Next() {
			var lib model.Package
			var name, description, normalizedName, homepage, repository string
			var createdAt, updatedAt int64

			if err := rows.Scan(&name, &lib.PurlType, &description, &normalizedName, &homepage, &repository, &createdAt, &updatedAt); err != nil {
				rows.Close()
				log.Err(err).Msg("failed to scan row")
				return err
			}

			// 清理无效的UTF-8字符
			if !utf8.ValidString(description) {
				description = strings.ToValidUTF8(description, "")
			}

			// 截断过长的 description
			if len(description) > 255 {
				description = description[:255]
			}

			lib.Name = name
			lib.Description = description
			lib.NormalizedName = normalizedName
			lib.HomepageURL = homepage
			lib.RepositoryURL = repository
			lib.CreatedAt = createdAt
			lib.UpdatedAt = updatedAt

			batch = append(batch, lib)
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		// 批量插入到目标数据库
		if err := dao.Transaction(func(tx *gorm.DB) error {
			for i := range batch {
				description := batch[i].Description
				// 清理无效的UTF-8字符
				if !utf8.ValidString(description) {
					description = strings.ToValidUTF8(description, "")
				}
				if len(description) > 255 {
					description = description[:255]
				}

				err := tx.Where("name = ? AND purl_type = ?", batch[i].Name, batch[i].PurlType).
					Assign(model.Package{
						ID:             model.NewID(),
						Description:    description,
						NormalizedName: batch[i].NormalizedName,
						HomepageURL:    batch[i].HomepageURL,
						RepositoryURL:  batch[i].RepositoryURL,
						UpdatedAt:      batch[i].UpdatedAt,
					}).
					FirstOrCreate(&batch[i]).Error
				if err != nil {
					fmt.Printf("导入失败: %s %s %s\n", batch[i].Name, batch[i].PurlType, err)
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}

		totalImported += len(batch)
		offset += importBatchSize
		fmt.Printf("已导入: %d / %d\n", totalImported, newRecordsCount)

		if len(batch) < importBatchSize {
			break
		}
	}

	fmt.Printf("导入完成！共导入 %d 条数据\n", totalImported)
	return nil
}

// importFromCSV 从 CSV 文件导入 crates 数据
func importFromCSV(csvPath string, existingLibs map[string]bool) error {
	// 检查文件是否存在
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		log.Err(err).Str("File", csvPath).Msg("CSV file not found")
		return err
	}

	// 创建临时 DuckDB 数据库
	tmpDbPath := filepath.Join(os.TempDir(), fmt.Sprintf("import_%d.duckdb", time.Now().UnixNano()))
	defer os.Remove(tmpDbPath)

	// 初始化 DuckDB
	db, err := sql.Open("duckdb", tmpDbPath)
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

	// 使用 COPY 命令导入 CSV 数据到 DuckDB（高性能）
	copySQL := fmt.Sprintf(`CREATE TABLE crates_import AS SELECT * FROM '%s';`, csvPath)

	fmt.Println("正在将 CSV 数据导入到 DuckDB...")
	startTime := time.Now()
	if _, err := db.Exec(copySQL); err != nil {
		log.Err(err).Msg("failed to copy CSV to DuckDB")
		return err
	}
	fmt.Printf("DuckDB 导入完成，耗时: %v\n", time.Since(startTime))

	// 获取导入记录数
	var totalCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM crates_import").Scan(&totalCount); err != nil {
		log.Err(err).Msg("failed to count records")
		return err
	}
	fmt.Printf("CSV 文件共 %d 条记录\n", totalCount)

	// 构建排除条件
	batchSize := 10000
	var excludeConditions []string
	keys := make([]string, 0, len(existingLibs))
	for key := range existingLibs {
		keys = append(keys, key)
	}

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := make([]string, 0, end-i)
		for j := i; j < end; j++ {
			batch = append(batch, fmt.Sprintf("'%s'", keys[j]))
		}
		excludeConditions = append(excludeConditions, strings.Join(batch, ","))
	}

	// 计算新增记录数
	var newRecordsCount int
	if len(excludeConditions) > 0 {
		excludeSQL := strings.Join(excludeConditions, ",")
		countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM crates_import WHERE name NOT IN (%s)`, excludeSQL)
		if err := db.QueryRow(countSQL).Scan(&newRecordsCount); err != nil {
			log.Err(err).Msg("failed to count new records")
			return err
		}
	} else {
		newRecordsCount = totalCount
	}

	fmt.Printf("新增 %d 条记录需要导入\n", newRecordsCount)

	if newRecordsCount == 0 {
		fmt.Println("没有新数据需要导入")
		return nil
	}

	// 批量导入数据
	fmt.Printf("正在批量导入数据到目标数据库...\n")
	importBatchSize := 1000
	totalImported := 0
	offset := 0

	// 构建查询 SQL - CSV 导入的临时表字段
	// DuckDB 导入 CSV 时 created_at/updated_at 会解析为 TIMESTAMP WITH TIME ZONE
	querySQL := `
		SELECT 
			name,
			'cargo' as purl_type,
			CASE WHEN description IS NULL THEN '' ELSE description END as description,
			LOWER(name) as normalized_name,
			CASE WHEN homepage IS NULL THEN '' ELSE homepage END as homepage,
			CASE WHEN repository IS NULL THEN '' ELSE repository END as repository,
			CAST(EPOCH(CAST(created_at AS TIMESTAMP)) AS BIGINT) as created_at,
			CAST(EPOCH(CAST(updated_at AS TIMESTAMP)) AS BIGINT) as updated_at
		FROM crates_import
	`

	if len(excludeConditions) > 0 {
		excludeSQL := strings.Join(excludeConditions, ",")
		querySQL = fmt.Sprintf("%s WHERE name NOT IN (%s)", querySQL, excludeSQL)
	}

	// 按批次处理数据
	for {
		batchSQL := fmt.Sprintf("%s LIMIT %d OFFSET %d", querySQL, importBatchSize, offset)
		rows, err := db.Query(batchSQL)
		if err != nil {
			log.Err(err).Msg("failed to query from DuckDB")
			return err
		}

		var batch []model.Package
		for rows.Next() {
			var lib model.Package
			var name, description, normalizedName, homepage, repository string
			var createdAt, updatedAt float64

			if err := rows.Scan(&name, &lib.PurlType, &description, &normalizedName, &homepage, &repository, &createdAt, &updatedAt); err != nil {
				rows.Close()
				log.Err(err).Msg("failed to scan row")
				return err
			}

			// 清理无效的UTF-8字符
			if !utf8.ValidString(description) {
				description = strings.ToValidUTF8(description, "")
			}

			// 截断过长的 description
			if len(description) > 255 {
				description = description[:255]
			}

			lib.Name = name
			lib.Description = description
			lib.NormalizedName = normalizedName
			lib.HomepageURL = homepage
			lib.RepositoryURL = repository
			lib.CreatedAt = int64(createdAt)
			lib.UpdatedAt = int64(updatedAt)

			batch = append(batch, lib)
		}
		rows.Close()

		if len(batch) == 0 {
			break
		}

		// 批量插入到目标数据库
		if err := dao.Transaction(func(tx *gorm.DB) error {
			for i := range batch {
				description := batch[i].Description
				// 清理无效的UTF-8字符
				if !utf8.ValidString(description) {
					description = strings.ToValidUTF8(description, "")
				}
				if len(description) > 255 {
					description = description[:255]
				}

				err := tx.Where("name = ? AND purl_type = ?", batch[i].Name, batch[i].PurlType).
					Assign(model.Package{
						Description:    description,
						NormalizedName: batch[i].NormalizedName,
						HomepageURL:    batch[i].HomepageURL,
						RepositoryURL:  batch[i].RepositoryURL,
						UpdatedAt:      batch[i].UpdatedAt,
					}).
					FirstOrCreate(&batch[i]).Error
				if err != nil {
					log.Err(err).Msgf("failed to insert: %#v", batch[i])
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}

		totalImported += len(batch)
		offset += importBatchSize
		fmt.Printf("已导入: %d / %d\n", totalImported, newRecordsCount)

		if len(batch) < importBatchSize {
			break
		}
	}

	fmt.Printf("导入完成！共导入 %d 条数据\n", totalImported)
	return nil
}
