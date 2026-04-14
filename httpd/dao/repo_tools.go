package dao

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alayou/techstack/model"
	"github.com/tmc/langchaingo/tools"
)

// ============ Field Names ============
const (
	FieldWhat           = "what"
	FieldPurpose        = "purpose"
	FieldValuePropose   = "value_propose"
	FieldQuickStart     = "quick_start"
	FieldTechStack      = "techstack"
	FieldCodeStructure  = "code_structure"
	FieldCodeRule       = "code_rule"
	FieldMainAPI        = "main_api"
	FieldUsageScenarios = "usage_scenarios"
	FieldStrength       = "strength"
	FieldWeakness       = "weakness"
	FieldSuitFor        = "suit_for"
	FieldNotSuitFor     = "not_suit_for"
)

// ============ Field Definitions ============
var repoFieldDefs = []struct {
	field   string
	descCN  string
	example string
}{
	{FieldWhat, "一句话概述", "这是一个用于处理HTTP请求的Go语言Web框架"},
	{FieldPurpose, "核心定位", "解决Web开发中的路由、中间件、上下文管理等核心问题"},
	{FieldValuePropose, "相比同类优势", "性能优异、API简洁、社区活跃、文档完善"},
	{FieldQuickStart, "快速开始", "go get github.com/example/pkg\n\nfunc main() {\n    r := New()\n    r.GET(\"/\", func(c *Context) {\n        c.String(\"Hello World\")\n    })\n    r.Run()\n}"},
	{FieldTechStack, "技术栈依赖", "Go 1.18+\n核心依赖:\n- gorilla/mux: 路由\n- golang-jwt/jwt: 认证"},
	{FieldCodeStructure, "项目目录结构", "├── cmd/          # 入口文件\n├── internal/     # 内部包\n├── pkg/          # 公共库\n└── api/          # API定义"},
	{FieldCodeRule, "编码规范", "遵循Uber Go编码规范\n- 错误处理: 使用fmt.Errorf包装\n- 命名: 驼峰式\n- 注释: 每个导出函数都需要文档注释"},
	{FieldMainAPI, "核心API", "New() *Engine\nGET(path, handler)\nPOST(path, handler)\nPUT(path, handler)\nDELETE(path, handler)\nUse(middleware)\nRun(addr)"},
	{FieldUsageScenarios, "最佳使用场景", "- 轻量级Web服务\n- RESTful API开发\n- 微服务网关\n- 前后端分离后端服务"},
	{FieldStrength, "优点", "- 性能优秀\n- API简洁易用\n- 中间件丰富\n- 文档完善\n- 社区活跃"},
	{FieldWeakness, "缺点限制", "- 相对较新，生态还在完善\n- 某些场景不如成熟框架稳定\n- 学习曲线较陡"},
	{FieldSuitFor, "适合项目", "- 中小型Web应用\n- API服务\n- 微服务\n- 需要高性能的场景"},
	{FieldNotSuitFor, "不适合项目", "- 超大型企业应用(考虑K8s集成)\n- 需要大量ORM功能的(考虑gorm)\n- 快速原型开发(考虑Ruby/Python)"},
}

// readRepoField 通用读字段工具
type readRepoField struct {
	repoId  int64
	field   string
	descCN  string
	example string
}

// Call implements tools.Tool
func (r *readRepoField) Call(ctx context.Context, input string) (out string, err error) {
	var result string
	err = gdb.Model(new(model.RepoTechAnalysis)).Select(r.field).Where("repo_id=?", r.repoId).Scan(&result).Error
	if err != nil {
		return "error: " + err.Error(), nil
	}
	return result, nil
}

// Description implements tools.Tool
func (r *readRepoField) Description() string {
	return fmt.Sprintf("读取字段 - %s。此工具不需要任何参数。", r.descCN)
}

// Name implements tools.Tool
func (r *readRepoField) Name() string {
	return "read_repo_" + r.field
}

// GetParameters 实现 ToolWithSchema 接口
func (r *readRepoField) GetParameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
}

var _ tools.Tool = (*readRepoField)(nil)
var _ ToolWithSchema = (*readRepoField)(nil)

// ==============================================
// Update Tool - 更新仓库字段
// ==============================================

// updateRepoField 通用更新字段工具
type updateRepoField struct {
	repoId int64
	field  string
	descCN string
}

// Call implements tools.Tool
func (u *updateRepoField) Call(ctx context.Context, input string) (out string, err error) {
	var content string
	params := make(map[string]any)
	if jsonErr := json.Unmarshal([]byte(input), &params); jsonErr == nil {
		if val, ok := params["content"].(string); ok {
			content = val
		}
	} else {
		content = input
	}

	if content == "" {
		return "error: content parameter is required", nil
	}

	err = gdb.Model(new(model.RepoTechAnalysis)).Where("repo_id=?", u.repoId).Update(u.field, content).Error
	if err != nil {
		return "error: " + err.Error(), nil
	}
	return "updated field " + u.descCN, nil
}

// Description implements tools.Tool
func (u *updateRepoField) Description() string {
	return fmt.Sprintf("更新 %s 字段。使用 'content' 参数指定新的内容。", u.descCN)
}

// Name implements tools.Tool
func (u *updateRepoField) Name() string {
	return "update_repo_" + u.field
}

// GetParameters 实现 ToolWithSchema 接口
func (u *updateRepoField) GetParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("新的 %s 内容", u.descCN),
			},
		},
		"required": []string{"content"},
	}
}

var _ tools.Tool = (*updateRepoField)(nil)
var _ ToolWithSchema = (*updateRepoField)(nil)

// ==============================================
// Tool Constructor Functions
// ==============================================

// NewRepoTools 创建仓库技术分析的所有工具
func NewRepoTools(repoId int64) []tools.Tool {
	toolsList := make([]tools.Tool, 0, len(repoFieldDefs)*2)

	for _, def := range repoFieldDefs {
		toolsList = append(toolsList, &readRepoField{
			repoId:  repoId,
			field:   def.field,
			descCN:  def.descCN,
			example: def.example,
		})

		toolsList = append(toolsList, &updateRepoField{
			repoId: repoId,
			field:  def.field,
			descCN: def.descCN,
		})
	}

	return toolsList
}

// NewReadRepoTools 仅创建读工具
func NewReadRepoTools(repoId int64) []tools.Tool {
	toolsList := make([]tools.Tool, 0, len(repoFieldDefs))

	for _, def := range repoFieldDefs {
		toolsList = append(toolsList, &readRepoField{
			repoId:  repoId,
			field:   def.field,
			descCN:  def.descCN,
			example: def.example,
		})
	}

	return toolsList
}

// NewUpdateRepoTools 仅创建写工具
func NewUpdateRepoTools(repoId int64) []tools.Tool {
	toolsList := make([]tools.Tool, 0, len(repoFieldDefs))

	for _, def := range repoFieldDefs {
		toolsList = append(toolsList, &updateRepoField{
			repoId: repoId,
			field:  def.field,
			descCN: def.descCN,
		})
	}

	return toolsList
}

// GetRepoFieldDefs 返回所有字段定义
func GetRepoFieldDefs() []struct {
	Field   string
	DescCN  string
	Example string
} {
	result := make([]struct {
		Field   string
		DescCN  string
		Example string
	}, len(repoFieldDefs))
	for i, def := range repoFieldDefs {
		result[i] = struct {
			Field   string
			DescCN  string
			Example string
		}{
			Field:   def.field,
			DescCN:  def.descCN,
			Example: def.example,
		}
	}
	return result
}

// ToolWithSchema 本地定义的接口，避免导入循环
type ToolWithSchema interface {
	tools.Tool
	GetParameters() map[string]any
}
