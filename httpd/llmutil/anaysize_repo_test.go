package llmutil

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/alayou/techstack/httpd/dao/memtools"
	"github.com/alayou/techstack/model"
	"github.com/alayou/techstack/pkg/repofs"
	"github.com/spf13/afero"
)

func getLlmModelConfig() (cfg model.LLMModelConfig, err error) {
	err = json.Unmarshal([]byte(`{
  "apiKey": "",
  "baseUrl": "https://api.minimaxi.com/v1",
  "provider": "openai",
  "model": "MiniMax-M2.7"
}`), &cfg)
	return
}

func TestDocAnalysize(t *testing.T) {
	t.SkipNow()
	raw, err := os.ReadFile("testdata/wiki.out")
	if err != nil {
		t.Fatal(err)
	}
	llmCfg, err := getLlmModelConfig()
	if err != nil {
		t.Fatal(err)
	}
	res, err := DocAnalysize(context.TODO(), &llmCfg, string(raw))
	if err != nil {
		t.Fatal(err)
	}

	// 验证返回结果的关键字段都已填充
	if res.What == "" {
		t.Error("What field should not be empty")
	}
	if res.Purpose == "" {
		t.Error("Purpose field should not be empty")
	}
	if res.ValuePropose == "" {
		t.Error("ValuePropose field should not be empty")
	}
	if res.QuickStart == "" {
		t.Error("QuickStart field should not be empty")
	}
	if res.TechStack == "" {
		t.Error("TechStack field should not be empty")
	}
	if res.CodeStructure == "" {
		t.Error("CodeStructure field should not be empty")
	}
	if res.CodeRule == "" {
		t.Error("CodeRule field should not be empty")
	}
	if res.MainAPI == "" {
		t.Error("MainAPI field should not be empty")
	}
	if res.UsageScenarios == "" {
		t.Error("UsageScenarios field should not be empty")
	}
	if res.Strength == "" {
		t.Error("Strength field should not be empty")
	}
	if res.Weakness == "" {
		t.Error("Weakness field should not be empty")
	}
	if res.SuitFor == "" {
		t.Error("SuitFor field should not be empty")
	}
	if res.NotSuitFor == "" {
		t.Error("NotSuitFor field should not be empty")
	}
}

const _rawContentMsg = "```json\n{\n  \"what\": \"techstackctl\",\"purpose\": \"核心定位\",\"techstack\": \"techstack\",\"value_propose\": \"相比同类库优势（AI总结）\",\"quick_start\": \"快速开始（可直接生成代码）\",\"code_structure\": \"技术栈、依赖、核心技术\",\"code_rule\": \"编码规范、接口风格、设计模式\",\"main_api\": \"核心API/接口/函数\",\"usage_scenarios\": \"最佳使用场景\",\"strength\": \"优点\",\"weakness\": \"缺点/限制\",\"suit_for\": \"适合什么项目\",\"not_suit_for\": \"1) Java；\\n2) 超大规模\\n3) 需要图形化界面；\\n4) 对 SLA 要求；\\n5) 希望开箱即用；\\n6) Redis 缓存层。\"\n}\n```"

func TestNormlizeContentJSON(t *testing.T) {
	t.SkipNow()
	raw := NormlizeContentJSON(_rawContentMsg)
	var res model.RepoTechAnalysis
	t.Log(raw[len(raw)-10:])
	err := json.Unmarshal([]byte(raw), &res)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(res)
	t.Log(string(a))
	// 验证返回结果的关键字段都已填充
	if res.What == "" {
		t.Error("What field should not be empty")
	}
	if res.Purpose == "" {
		t.Error("Purpose field should not be empty")
	}
	if res.ValuePropose == "" {
		t.Error("ValuePropose field should not be empty")
	}
	if res.QuickStart == "" {
		t.Error("QuickStart field should not be empty")
	}
	if res.TechStack == "" {
		t.Error("TechStack field should not be empty")
	}
	if res.CodeStructure == "" {
		t.Error("CodeStructure field should not be empty")
	}
	if res.CodeRule == "" {
		t.Error("CodeRule field should not be empty")
	}
	if res.MainAPI == "" {
		t.Error("MainAPI field should not be empty")
	}
	if res.UsageScenarios == "" {
		t.Error("UsageScenarios field should not be empty")
	}
	if res.Strength == "" {
		t.Error("Strength field should not be empty")
	}
	if res.Weakness == "" {
		t.Error("Weakness field should not be empty")
	}
	if res.SuitFor == "" {
		t.Error("SuitFor field should not be empty")
	}
	if res.NotSuitFor == "" {
		t.Error("NotSuitFor field should not be empty")
	}
}

func TestAgentAnalysizeRepo(t *testing.T) {
	llmCfg, err := getLlmModelConfig()
	if err != nil {
		t.Fatal(err)
	}
	osfs := afero.NewBasePathFs(afero.NewOsFs(), "/Users/const/Space/wuxia/techstack")
	tools := repofs.NewLLMFsTools(osfs)
	var result = &model.RepoTechAnalysis{}
	tools2 := memtools.NewRepoTools(result)
	files := ""
	for _, t := range tools {
		if t.Name() == "list_files" {
			files, _ = t.Call(context.TODO(), "")
		}
	}
	output, err := AgentAnalysizeRepo(t.Context(), &llmCfg, files, append(tools, tools2...))
	if err != nil {
		t.Fatal(err)
	}
	t.Error(result)
	t.Log(output)
}
