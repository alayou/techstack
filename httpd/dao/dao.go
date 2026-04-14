package dao

import (
	"fmt"
	"time"

	"github.com/gorpher/gone/gormutil"

	"github.com/alayou/techstack/global"
	"gorm.io/gorm"

	"github.com/alayou/techstack/model"
)

var gdb *gorm.DB

const logSender = "dao"

func Ready() bool {
	return gdb != nil
}

func Init(driver, dsn string) error {
	conf := gormutil.Config{
		Driver: driver,
		DSN:    dsn,
	}
	var (
		db  *gorm.DB
		err error
	)
	db, err = gormutil.New(global.Config.Debug, conf)
	if err != nil {
		return err
	}

	if global.Config.Debug {
		gdb = db.Debug()
	} else {
		gdb = db
	}

	if err = gdb.AutoMigrate(model.Tables()...); err != nil {
		return err
	}

	return nil
}

// Transaction 开启事物
func Transaction(fc func(tx *gorm.DB) error) error {
	return gdb.Transaction(fc)
}

// View 查询
func View(fc func(tx *gorm.DB) error) error {
	return fc(gdb)
}

// FormatTimestampToIsoDatetime 格式化时间戳为字符日期格式
func FormatTimestampToIsoDatetime(db *gorm.DB, filed string) string {
	name := db.Name()
	if name == "postgres" {
		return fmt.Sprintf("to_char(to_timestamp(%s), 'yyyy-MM-dd')", filed)
	}
	if name == "sqlite" || name == "sqlite3" {
		return "strftime('%Y-%m-%d',datetime(" + filed + ",'unixepoch','localtime'))"
	}
	if name == "mysql" {
		return "from_unixtime(" + filed + ",'%Y-%m-%d')"
	}
	return filed
}

// GetFormatCurrentDate 获取当前日期
func GetFormatCurrentDate(db *gorm.DB) string {
	name := db.Name()
	if name == "postgres" {
		return "CURRENT_DATE::TEXT"
	}
	if name == "sqlite" || name == "sqlite3" {
		return "date()"
	}
	if name == "mysql" {
		return "CURRENT_DATE()"
	}
	return ""
}

func WithPage(pageNo, pageSize int64) (offset, limit int) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset = int((pageNo - 1) * pageSize)
	limit = int(pageSize)
	return
}

func WithDateRange(start, end int64) (int64, int64) {
	now := time.Now().UTC().Unix()
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = 0
	}
	if start > end {
		start = end
	}
	if end > now {
		end = now
	}
	if start > now {
		start = 0
	}
	return start, end
}
