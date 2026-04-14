package llmutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/alayou/techstack/model"
	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
)

// toolLoggingHandler 自定义回调处理器，用于记录工具调用信息
type toolLoggingHandler struct{}

func (h *toolLoggingHandler) HandleText(ctx context.Context, text string) {}

func (h *toolLoggingHandler) HandleLLMStart(ctx context.Context, prompts []string) {
	log.Info().Int("prompt_count", len(prompts)).Msg("LLM 开始生成")
}

func (h *toolLoggingHandler) HandleLLMGenerateContentStart(ctx context.Context, ms []llms.MessageContent) {
	log.Info().Int("message_count", len(ms)).Msg("LLM 开始生成内容")
}

func (h *toolLoggingHandler) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	log.Info().Int("choice_count", len(res.Choices)).Msg("LLM 内容生成结束")
}

func (h *toolLoggingHandler) HandleLLMError(ctx context.Context, err error) {
	log.Error().Err(err).Msg("LLM 错误")
}

func (h *toolLoggingHandler) HandleChainStart(ctx context.Context, inputs map[string]any) {
	log.Info().Msg("Chain 开始执行")
}

func (h *toolLoggingHandler) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	log.Info().Interface("outputs", outputs).Msg("Chain 执行结束")
}

func (h *toolLoggingHandler) HandleChainError(ctx context.Context, err error) {
	log.Error().Err(err).Msg("Chain 错误")
}

func (h *toolLoggingHandler) HandleToolStart(ctx context.Context, input string) {
	log.Info().Str("input", input).Msg("工具调用开始")
}

func (h *toolLoggingHandler) HandleToolEnd(ctx context.Context, output string) {
	log.Info().Str("output", output).Msg("工具调用结束")
}

func (h *toolLoggingHandler) HandleToolError(ctx context.Context, err error) {
	log.Error().Err(err).Msg("工具调用错误")
}

func (h *toolLoggingHandler) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	log.Info().
		Str("tool", action.Tool).
		Str("tool_input", action.ToolInput).
		Msg("Agent 执行动作")
}

func (h *toolLoggingHandler) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	log.Info().Interface("return_values", finish.ReturnValues).Msg("Agent 完成")
}

func (h *toolLoggingHandler) HandleRetrieverStart(ctx context.Context, query string) {}

func (h *toolLoggingHandler) HandleRetrieverEnd(ctx context.Context, query string, documents []schema.Document) {
}

func (h *toolLoggingHandler) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	log.Info().Msg(string(chunk))
}

const MaxChar = 68000

func DocAnalysize(ctx context.Context, llmConfig *model.LLMModelConfig, doc string) (result model.RepoTechAnalysis, err error) {
	// 2. 送给大模型，要求返回 JSON 格式的 RepoTechAnalysis
	prompt := fmt.Sprintf(`
你是一个开源项目架构分析专家，请根据项目文档生成完整的技术分析报告。
what: 一句话概述：这个库是做什么的
purpose: 核心定位、解决什么问题
value_propose:相比同类库优势（AI总结）
quick_start:快速开始（可直接生成代码）
techstack:技术栈、依赖、核心技术
code_structure:项目目录结构说明
code_rule:编码规范、接口风格、设计模式
main_api:核心API/接口/函数
usage_scenarios:最佳使用场景
strength:优点
weakness:缺点/限制
suit_for:适合什么项目,适合完成什么
not_suit_for:不适合什么项目

输出必须是严格的 JSON，不要其他内容。
JSON 结构：
{
	"what": "...", 
	"purpose": "...",
	"value_propose": "...",
	"quick_start": "...",
	"techstack": "...",
	"code_structure": "...",
	"code_rule": "...",
	"main_api": "...",
	"usage_scenarios": "...",
	"strength": "...",
	"weakness": "...",
	"suit_for": "...",
	"not_suit_for": "..."
}

文档：
%s
`, string(doc))
	if len(prompt) > MaxChar {
		prompt = prompt[:MaxChar]
	}
	var llm *openai.LLM

	llm, err = NewOpenAIClient(llmConfig)
	if err != nil {
		return
	}
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeGeneric,
			Parts: []llms.ContentPart{
				llms.TextPart(prompt),
			},
		},
	}

	var resp *llms.ContentResponse
	resp, err = llm.GenerateContent(ctx, messages,
		llms.WithTemperature(0.3),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			log.Info().Msg(string(chunk))
			return nil
		}))
	if err != nil {
		return
	}

	if len(resp.Choices) == 0 {
		err = errors.New("无输出内容")
		return
	}
	content := NormlizeContentJSON(strings.ToLower(resp.Choices[0].Content))
	log.Trace().Str("Body", content).Msg("调试Agent 返回结果")
	// 3. 解析 JSON 到结构体
	if err = json.Unmarshal([]byte(content), &result); err != nil {
		log.Error().Str("conntent", content).Msg("AI返回格式错误")
		err = fmt.Errorf("AI返回格式错误: %v", err)
		return
	}
	return
}

func NormlizeContentJSON(content string) string {
	if content == "" {
		return ""
	}
	firstIndx := strings.Index(content, "{")
	lastIndx := strings.LastIndex(content, "}")
	if firstIndx < lastIndx {
		return content[firstIndx : lastIndx+1]
	}
	return content
}

// extraBodyTransport 注入 extra_body
type extraBodyTransport struct {
	base  http.RoundTripper
	extra map[string]any
}

func (t *extraBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		return t.base.RoundTrip(req)
	}

	// 1. 读取原请求体
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	// 2. 解析为 map
	var data map[string]any
	if err := json.Unmarshal(body, &data); err == nil {
		// 3. 注入 extra_body
		maps.Copy(data, t.extra)
		// 4. 重新序列化
		newBody, _ := json.Marshal(data)
		req.Body = io.NopCloser(bytes.NewBuffer(newBody))
		req.ContentLength = int64(len(newBody))
		log.Trace().Str("Body", string(newBody)).Msg("打印 Agent Body")
	}

	return t.base.RoundTrip(req)
}

func NewOpenAIClient(llmConfig *model.LLMModelConfig) (*openai.LLM, error) {
	var opts = []openai.Option{
		openai.WithBaseURL(llmConfig.BaseUrl),
		openai.WithToken(llmConfig.ApiKey),
		openai.WithModel(llmConfig.Model),
	}
	if strings.HasPrefix(llmConfig.BaseUrl, "https://api.minimaxi.com/v1") {
		opts = append(opts, openai.WithHTTPClient(&http.Client{
			Timeout: 3 * time.Minute,
			Transport: &extraBodyTransport{
				base: http.DefaultTransport,
				extra: map[string]any{
					"reasoning_split": true,
				},
			},
		}))
	}
	return openai.New(
		opts...,
	)
}

func AgentAnalysizeRepo(ctx context.Context, llmConfig *model.LLMModelConfig, repoDescrbe string, extraTools []tools.Tool) (output string, err error) {
	var llm *openai.LLM
	prompt := fmt.Sprintf(`
你是开源项目架构分析师。请使用工具读取代码与文档，生成一份标准技术分析报告。
必须使用工具列表，调用合适的工具，来存储相关信息报告信息。
报告必须包含以下内容，客观真实、基于源码、不编造：
项目用途、解决问题、优势亮点、快速上手、技术栈、目录结构、代码规范、核心API、适用场景、优缺点、适合/不适合项目。
%s

`, repoDescrbe)
	llm, err = NewOpenAIClient(llmConfig)
	if err != nil {
		return
	}
	// 使用支持自定义参数 schema 的 agent
	agent := NewOpenAIFunctionsAgentWithSchema(llm, extraTools)

	// 创建自定义回调处理器
	callbackHandler := &toolLoggingHandler{}
	executor := agents.NewExecutor(agent, agents.WithCallbacksHandler(callbackHandler), agents.WithMaxIterations(500))
	return chains.Run(ctx, executor, prompt, chains.WithCallback(callbackHandler))
}
