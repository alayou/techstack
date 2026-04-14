package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/gorpher/gone"
	"github.com/gorpher/gone/osutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const (
	SettingTableName                 = "t_setting"
	BackgroundTaskTableName          = "t_bg_tasks"
	PackageTableName                 = "t_packages"
	PackageVersionTableName          = "t_package_versions"
	RepoPackageIndexTableName        = "t_repo_pkg_index"
	RepoPackageIndexVersionTableName = "t_repo_pkg_version_index"

	PublicRepoTableName       = "t_repo_public"
	RepoDependencyTableName   = "t_repo_dependency"
	RepoTechAnalysisTableName = "t_repo_tech_analysis"

	UserTableName         = "t_users"
	UserRepoStarTableName = "t_user_stars"
	UserApiKeyTableName   = "t_user_keys"
)

func Tables() []interface{} {
	return []interface{}{
		new(Setting),

		new(BackgroundTask),
		new(Package),
		new(PackageVersion),

		new(PublicRepo),
		new(RepoDependency),
		new(RepoPkgIndex),
		new(RepoPkgVersionIndex),
		new(RepoTechAnalysis),

		new(User),
		new(UserRepoStar),

		new(UserApiKey),
	}
}

type StringArray []string

func (j *StringArray) Scan(value interface{}) error {
	var bytes []byte
	str, ok := value.(string)
	if ok {
		bytes = []byte(str)
	}
	if !ok {
		bytes, ok = value.([]byte)
		if !ok {
			return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
		}
	}
	result := StringArray{}
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j StringArray) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	bys, err := json.Marshal(j)
	return string(bys), err
}

func (j StringArray) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(j))
}

func (j *StringArray) UnmarshalJSON(b []byte) error {
	var (
		t   = make([]string, 0)
		err error
		in  string
	)
	err = json.Unmarshal(b, &t)
	if err == nil {
		*j = t
		return nil
	}
	var f []int64
	err = json.Unmarshal(b, &f)
	if err != nil {
		return err
	}

	t = make([]string, len(f))
	for i, v := range f {
		in = strconv.FormatInt(v, 10)
		t[i] = in
	}
	*j = t
	return nil
}

type IntArray []int64

func (j *IntArray) Scan(value interface{}) error {
	var bytes []byte
	str, ok := value.(string)
	if ok {
		bytes = []byte(str)
	}
	if !ok {
		bytes, ok = value.([]byte)
		if !ok {
			return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
		}
	}
	result := IntArray{}
	err := json.Unmarshal(bytes, &result)
	*j = result
	return err
}

func (j IntArray) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	bys, err := json.Marshal([]int64(j))
	return string(bys), err
}

func (j IntArray) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal([]int64(j))
}

func (j *IntArray) UnmarshalJSON(b []byte) error {
	var (
		t   = make([]int64, 0)
		err error
		in  int64
	)
	err = json.Unmarshal(b, &t)
	if err == nil {
		*j = t
		return nil
	}
	var f []string
	err = json.Unmarshal(b, &f)
	if err != nil {
		return err
	}

	t = make([]int64, len(f))
	for i, v := range f {
		in, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return err
		}
		t[i] = in
	}
	*j = t
	return err
}

func (j *IntArray) Int64() []int64 {
	return *j
}

func NewID() ID {
	return ID(osutil.ID.SInt64())
}

type ID int64

func (s ID) String() string {
	return strconv.FormatInt(int64(s), 10)
}
func (s ID) Int64() int64 {
	return int64(s)
}

func (s ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(s), 10))
}

func (s *ID) UnmarshalJSON(b []byte) error {
	var (
		err error
		in  int64
	)
	err = json.Unmarshal(b, &in)
	if err == nil {
		*s = ID(in)
		return nil
	}
	var v string
	err = json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	in, err = strconv.ParseInt(v, 10, 64)
	if err != nil {
		return err
	}
	*s = ID(in)
	return nil
}

func (ID) CreateClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{IDCreateClauses{Field: f}}
}

type IDCreateClauses struct {
	Field *schema.Field
}

func (s IDCreateClauses) Name() string {
	return ""
}

func (s IDCreateClauses) Build(clause.Builder) {
}

func (s IDCreateClauses) MergeClause(*clause.Clause) {
}

func (s IDCreateClauses) ModifyStatement(stmt *gorm.Statement) {
	var setValue = func(field *schema.Field, value reflect.Value) {
		_, zero := field.ValueOf(stmt.Context, value)
		if zero {
			// 这个ID 目前支持结构体自动添加uuid
			uuid := gone.NumberID()
			stmt.SetColumn(s.Field.DBName, uuid, true)
		}

	}
	if stmt.Schema != nil {
		if field := stmt.Schema.LookUpField(s.Field.Name); field != nil {
			switch stmt.ReflectValue.Kind() { //nolint
			case reflect.Slice, reflect.Array:
				// TODO  批量创建自动生成UUID
			default:
				setValue(field, stmt.ReflectValue)
			}
		}
	}
}

// Opts defiend JSON data type, need to implements driver.Valuer, sql.Scanner interface
type Opts map[string]interface{}

func (m Opts) GetString(name string) string {
	if v, ok := m[name].(string); ok {
		return v
	}

	return ""
}

func (m Opts) GetBool(name string) bool {
	if v, ok := m[name].(bool); ok {
		return v
	}

	return false
}

func (m Opts) GetInt(name string) int {
	if v, ok := m[name].(int); ok {
		return v
	}
	if v, ok := m[name].(int64); ok {
		return int(v)
	}
	if v, ok := m[name].(float64); ok {
		return int(v)
	}

	return 0
}

func (m Opts) GetSlice(name string) []string {
	if v, ok := m[name].([]string); ok {
		return v
	}
	if v, ok := m[name].(string); ok {
		return strings.Split(v, ",")
	}
	return []string{}
}

// Value return json value, implement driver.Valuer interface
func (m Opts) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	ba, err := m.MarshalJSON()
	return string(ba), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *Opts) Scan(val interface{}) error {
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := map[string]interface{}{}
	err := json.Unmarshal(ba, &t)
	*m = Opts(t)
	return err
}

// MarshalJSON to output non base64 encoded []byte
func (m Opts) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	t := (map[string]interface{})(m)
	return json.Marshal(t)
}

// UnmarshalJSON to deserialize []byte
func (m *Opts) UnmarshalJSON(b []byte) error {
	t := map[string]interface{}{}
	err := json.Unmarshal(b, &t)
	*m = Opts(t)
	return err
}

// GormDataType gorm common data type
func (m Opts) GormDataType() string {
	return "jsonmap"
}

// GormDBDataType gorm db data type
func (Opts) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "TEXT"
	case "mysql":
		return "TEXT"
	case "postgres":
		return "TEXT"
	}
	return ""
}
