package model

import (
	"fmt"
	"strings"
)

const (
	RepoStatusWaiting = "waiting" // 未开始，等待中，需要触发条件或手动开始
	RepoStatusPending = "pending"
	RepoStatusRunning = "running"
	RepoStatusSuccess = "success"
	RepoStatusFailed  = "failed"
)

type PublicRepo struct {
	ID              ID     `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"`
	FullName        string `gorm:"uniqueIndex" json:"full_name"`                  // owner/name
	RepoURL         string `gorm:"size:512;not null;uniqueIndex" json:"repo_url"` // Git 仓库地址
	RepoName        string `gorm:"size:255;not null" json:"repo_name"`            // 仓库名称
	DefaultBranch   string `gorm:"size:100" json:"default_branch"`                // 默认分支：main/master
	Description     string `json:"description"`
	Stars           int64  `json:"stars"`
	Forks           int64  `json:"forks"`
	Language        string `json:"language"`
	License         string `json:"license"`
	ImportStatus    string `gorm:"column:import_status;size:50;default:pending" json:"import_status"`     // 整体解析状态
	AnalysisStatus  string `gorm:"column:analysis_status;size:50;default:pending" json:"analysis_status"` // 整体解析状态
	AnalysisSummary string `gorm:"column:analysis_summary;type:text" json:"analysis_summary"`             // 分析总结
	LastAnalyzedAt  int64  `gorm:"column:last_analyzed_at" json:"last_analyzed_at"`                       // 最后一次解析时间
	CreatedAt       int64  `json:"created_at" gorm:"autoCreateTime;not null"`                             // 创建时间
	UpdatedAt       int64  `json:"updated_at" gorm:"autoUpdateTime;not null"`                             // 更新时间

	Dependencies  []RepoDependency `json:"dependencies" gorm:"-"`  // 依赖库
	TechAnanlysis RepoTechAnalysis `json:"tech_analysis" gorm:"-"` // 技术分析
}

func (s *PublicRepo) FormtString() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "repo_url: %s\n", s.RepoURL)
	fmt.Fprintf(sb, "repo_name: %s\n", s.RepoName)
	fmt.Fprintf(sb, "description: %s\n", s.Description)
	fmt.Fprintf(sb, "stars: %d\n", s.Stars)
	fmt.Fprintf(sb, "forks: %d\n", s.Forks)
	fmt.Fprintf(sb, "language: %s\n", s.Language)
	fmt.Fprintf(sb, "license: %s\n", s.License)
	return sb.String()
}

func (PublicRepo) TableName() string {
	return PublicRepoTableName
}

// RepoDependency
type RepoDependency struct {
	ID         ID     `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"` // 主键ID
	RepoType   string `json:"repo_type"`                                                 // user,public 关联用户仓库还是开放仓库
	RepoID     int64  `json:"repo_id,string,omitempty" gorm:"index;uniqueIndex:idx_pub_dep_purl"`
	PURL       string `gorm:"column:purl;size:512;not null;uniqueIndex:idx_pub_dep_purl" json:"purl"` // 依赖PURL
	Version    string `gorm:"size:100;not null" json:"version"`                                       // 版本号
	SourceFile string `gorm:"size:512;not null" json:"source_file"`                                   // 来源文件：package.json/go.mod
}

func (RepoDependency) TableName() string {
	return RepoDependencyTableName
}

type RepoTechAnalysis struct {
	ID         ID     `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"`
	RepoType   string `json:"repo_type"` // user,public 关联用户仓库还是开放仓库
	RepoID     int64  `json:"repo_id,string,omitempty" gorm:"index;uniqueIndex:idx_pub_dep_purl"`
	CommitHash string `json:"commit_hash" gorm:"type:varchar(128);"`
	Branch     string `json:"branch" gorm:"type:varchar(128);"`

	What           string `json:"what" gorm:"column:what:text"`                            // 一句话概述：这个库是做什么的
	Purpose        string `json:"purpose" gorm:"column:purpose:text"`                      // 核心定位、解决什么问题
	ValuePropose   string `json:"value_propose" gorm:"column:value_propose;type:text"`     // 相比同类库优势（AI总结）
	QuickStart     string `json:"quick_start" gorm:"column:quick_start;type:text"`         // 快速开始（可直接生成代码）
	TechStack      string `json:"techstack" gorm:"column:techstack;type:text"`             // 技术栈、依赖、核心技术
	CodeStructure  string `json:"code_structure" gorm:"column:code_structure;type:text"`   // 项目目录结构说明
	CodeRule       string `json:"code_rule" gorm:"column:code_rule;type:text"`             // 编码规范、接口风格、设计模式
	MainAPI        string `json:"main_api" gorm:"column:main_api;type:text"`               // 核心API/接口/函数
	UsageScenarios string `json:"usage_scenarios" gorm:"column:usage_scenarios;type:text"` // 最佳使用场景
	Strength       string `json:"strength" gorm:"column:strength:text"`                    // 优点
	Weakness       string `json:"weakness" gorm:"column:weakness:text"`                    // 缺点/限制
	SuitFor        string `json:"suit_for" gorm:"column:suit_for;type:text"`               // 适合什么项目
	NotSuitFor     string `json:"not_suit_for" gorm:"column:not_suit_for;type:text"`       // 不适合什么项目

	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime;not null"` // 创建时间
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime;not null"` // 更新时间
}

func (RepoTechAnalysis) TableName() string {
	return RepoTechAnalysisTableName
}

type RepoPkgIndex struct {
	RepoID int64 `json:"repo_id" gorm:"column:repo_id;uniqueIndex:idx_rep_dep_pkg"` // 仓库ID
	DepID  int64 `json:"dep_id" gorm:"column:dep_id;uniqueIndex:idx_rep_dep_pkg"`   // 仓库依赖ID
	PkgID  int64 `json:"pkg_id" gorm:"column:pkg_id;uniqueIndex:idx_rep_dep_pkg"`   // 包ID
}

func (RepoPkgIndex) TableName() string {
	return RepoPackageIndexTableName
}

type RepoPkgVersionIndex struct {
	RepoID int64 `json:"repo_id" gorm:"column:repo_id;uniqueIndex:idx_rep_dep_pkg_version"` // 仓库ID
	DepID  int64 `json:"dep_id" gorm:"column:dep_id;uniqueIndex:idx_rep_dep_pkg_version"`   // 仓库依赖ID
	PkgID  int64 `json:"pkg_id" gorm:"column:pkg_id;uniqueIndex:idx_rep_dep_pkg_version"`   // 包版本ID
}

func (RepoPkgVersionIndex) TableName() string {
	return RepoPackageIndexVersionTableName
}
