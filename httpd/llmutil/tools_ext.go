package llmutil

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
)

// agentScratchpad "agent_scratchpad" for the agent to put its thoughts in.
const agentScratchpad = "agent_scratchpad"

// ToolWithSchema 扩展工具接口，支持自定义参数 schema
type ToolWithSchema interface {
	tools.Tool
	// GetParameters 返回工具的参数定义（JSON Schema 格式）
	GetParameters() map[string]any
}

// OpenAIFunctionsAgentWithSchema 支持自定义参数 schema 的 agent
type OpenAIFunctionsAgentWithSchema struct {
	LLM              llms.Model
	Prompt           prompts.FormatPrompter
	Tools            []tools.Tool
	OutputKey        string
	CallbacksHandler callbacks.Handler
}

var _ agents.Agent = (*OpenAIFunctionsAgentWithSchema)(nil)

// NewOpenAIFunctionsAgentWithSchema 创建支持自定义参数 schema 的 agent
func NewOpenAIFunctionsAgentWithSchema(llm llms.Model, tools []tools.Tool, opts ...agents.Option) *OpenAIFunctionsAgentWithSchema {
	// 创建一个简单的选项处理
	var systemMessage = "You are a helpful AI assistant."
	var outputKey = "output"
	var callbacksHandler callbacks.Handler
	var extraMessages []prompts.MessageFormatter

	// 处理选项 - 我们需要手动处理，因为内部选项不可导出
	// 对于这个实现，我们只创建基本版本

	return &OpenAIFunctionsAgentWithSchema{
		LLM: llm,
		Prompt: createOpenAIFunctionPromptWithSchema(
			systemMessage,
			extraMessages,
		),
		Tools:            tools,
		OutputKey:        outputKey,
		CallbacksHandler: callbacksHandler,
	}
}

// functions 生成函数定义，支持自定义参数 schema
func (o *OpenAIFunctionsAgentWithSchema) functions() []llms.FunctionDefinition {
	res := make([]llms.FunctionDefinition, 0)
	for _, tool := range o.Tools {
		fnDef := llms.FunctionDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
		}

		// 检查工具是否支持自定义参数 schema
		if toolWithSchema, ok := tool.(ToolWithSchema); ok {
			fnDef.Parameters = toolWithSchema.GetParameters()
		} else {
			// 默认使用原来的 __arg1 模式
			fnDef.Parameters = map[string]any{
				"properties": map[string]any{
					"__arg1": map[string]string{"title": "__arg1", "type": "string"},
				},
				"required": []string{"__arg1"},
				"type":     "object",
			}
		}

		res = append(res, fnDef)
	}
	return res
}

// Plan 实现 Agent 接口的 Plan 方法
func (o *OpenAIFunctionsAgentWithSchema) Plan(
	ctx context.Context,
	intermediateSteps []schema.AgentStep,
	inputs map[string]string,
	options ...chains.ChainCallOption,
) ([]schema.AgentAction, *schema.AgentFinish, error) {
	fullInputs := make(map[string]any, len(inputs))
	for key, value := range inputs {
		fullInputs[key] = value
	}
	fullInputs[agentScratchpad] = o.constructScratchPad(intermediateSteps)

	var stream func(ctx context.Context, chunk []byte) error

	if o.CallbacksHandler != nil {
		stream = func(ctx context.Context, chunk []byte) error {
			o.CallbacksHandler.HandleStreamingFunc(ctx, chunk)
			return nil
		}
	}

	prompt, err := o.Prompt.FormatPrompt(fullInputs)
	if err != nil {
		return nil, nil, err
	}

	mcList := make([]llms.MessageContent, len(prompt.Messages()))
	for i, msg := range prompt.Messages() {
		role := msg.GetType()
		text := msg.GetContent()

		var mc llms.MessageContent

		switch p := msg.(type) {
		case llms.ToolChatMessage:
			mc = llms.MessageContent{
				Role: role,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: p.ID,
					Content:    p.Content,
				}},
			}

		case llms.FunctionChatMessage:
			mc = llms.MessageContent{
				Role: role,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					Name:    p.Name,
					Content: p.Content,
				}},
			}

		case llms.AIChatMessage:
			if len(p.ToolCalls) > 0 {
				toolCallParts := make([]llms.ContentPart, 0, len(p.ToolCalls))
				for _, tc := range p.ToolCalls {
					toolCallParts = append(toolCallParts, llms.ToolCall{
						ID:           tc.ID,
						Type:         tc.Type,
						FunctionCall: tc.FunctionCall,
					})
				}
				mc = llms.MessageContent{
					Role:  role,
					Parts: toolCallParts,
				}
			} else {
				mc = llms.MessageContent{
					Role:  role,
					Parts: []llms.ContentPart{llms.TextContent{Text: text}},
				}
			}
		default:
			mc = llms.MessageContent{
				Role:  role,
				Parts: []llms.ContentPart{llms.TextContent{Text: text}},
			}
		}
		mcList[i] = mc
	}

	// Build LLM call options, including user-provided options
	llmOptions := []llms.CallOption{llms.WithFunctions(o.functions()), llms.WithStreamingFunc(stream)}
	llmOptions = append(llmOptions, chains.GetLLMCallOptions(options...)...)

	result, err := o.LLM.GenerateContent(ctx, mcList, llmOptions...)
	if err != nil {
		return nil, nil, err
	}

	return o.ParseOutput(result)
}

// GetInputKeys 实现 Agent 接口
func (o *OpenAIFunctionsAgentWithSchema) GetInputKeys() []string {
	chainInputs := o.Prompt.GetInputVariables()

	// Remove inputs given in plan.
	agentInput := make([]string, 0, len(chainInputs))
	for _, v := range chainInputs {
		if v == agentScratchpad {
			continue
		}
		agentInput = append(agentInput, v)
	}

	return agentInput
}

// GetOutputKeys 实现 Agent 接口
func (o *OpenAIFunctionsAgentWithSchema) GetOutputKeys() []string {
	return []string{o.OutputKey}
}

// GetTools 实现 Agent 接口
func (o *OpenAIFunctionsAgentWithSchema) GetTools() []tools.Tool {
	return o.Tools
}

// constructScratchPad 构建 agent 工作区
func (o *OpenAIFunctionsAgentWithSchema) constructScratchPad(steps []schema.AgentStep) []llms.ChatMessage {
	if len(steps) == 0 {
		return nil
	}

	messages := make([]llms.ChatMessage, 0)

	var currentToolCalls []llms.ToolCall
	var currentLog string

	for i, step := range steps {
		if i == 0 || step.Action.Log != steps[i-1].Action.Log {
			if len(currentToolCalls) > 0 {
				messages = append(messages, llms.AIChatMessage{
					Content:   currentLog,
					ToolCalls: currentToolCalls,
				})
				for j := i - len(currentToolCalls); j < i; j++ {
					messages = append(messages, llms.ToolChatMessage{
						ID:      steps[j].Action.ToolID,
						Content: steps[j].Observation,
					})
				}
				currentToolCalls = nil
			}
			currentLog = step.Action.Log
		}

		currentToolCalls = append(currentToolCalls, llms.ToolCall{
			ID:   step.Action.ToolID,
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      step.Action.Tool,
				Arguments: step.Action.ToolInput,
			},
		})
	}

	if len(currentToolCalls) > 0 {
		messages = append(messages, llms.AIChatMessage{
			Content:   currentLog,
			ToolCalls: currentToolCalls,
		})
		for j := len(steps) - len(currentToolCalls); j < len(steps); j++ {
			messages = append(messages, llms.ToolChatMessage{
				ID:      steps[j].Action.ToolID,
				Content: steps[j].Observation,
			})
		}
	}

	return messages
}

// ParseOutput 解析输出
func (o *OpenAIFunctionsAgentWithSchema) ParseOutput(contentResp *llms.ContentResponse) (
	[]schema.AgentAction, *schema.AgentFinish, error,
) {
	if contentResp == nil || len(contentResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("no choices in response")
	}
	choice := contentResp.Choices[0]

	if len(choice.ToolCalls) > 0 {
		actions := make([]schema.AgentAction, 0, len(choice.ToolCalls))

		for _, toolCall := range choice.ToolCalls {
			functionName := toolCall.FunctionCall.Name
			toolInputStr := toolCall.FunctionCall.Arguments
			toolInputMap := make(map[string]any, 0)
			err := json.Unmarshal([]byte(toolInputStr), &toolInputMap)

			toolInput := toolInputStr
			if err == nil {
				// 查找对应的工具，看是否需要特殊处理输入
				var targetTool tools.Tool
				for _, t := range o.Tools {
					if t.Name() == functionName {
						targetTool = t
						break
					}
				}

				// 如果工具支持自定义 schema，直接将整个 JSON 作为输入
				if _, ok := targetTool.(ToolWithSchema); ok {
					toolInput = toolInputStr
				} else if arg1, ok := toolInputMap["__arg1"]; ok {
					// 兼容旧模式
					toolInputCheck, ok := arg1.(string)
					if ok {
						toolInput = toolInputCheck
					}
				}
			}

			contentMsg := "\n"
			if choice.Content != "" {
				contentMsg = fmt.Sprintf("responded: %s\n", choice.Content)
			}

			actions = append(actions, schema.AgentAction{
				Tool:      functionName,
				ToolInput: toolInput,
				Log:       fmt.Sprintf("Invoking: %s with %s %s", functionName, toolInputStr, contentMsg),
				ToolID:    toolCall.ID,
			})
		}

		return actions, nil, nil
	}

	if choice.FuncCall != nil {
		functionCall := choice.FuncCall
		functionName := functionCall.Name
		toolInputStr := functionCall.Arguments
		toolInputMap := make(map[string]any, 0)
		err := json.Unmarshal([]byte(toolInputStr), &toolInputMap)
		if err != nil {
			return []schema.AgentAction{
				{
					Tool:      functionName,
					ToolInput: toolInputStr,
					Log:       fmt.Sprintf("Invoking: %s with %s\n", functionName, toolInputStr),
					ToolID:    "",
				},
			}, nil, nil
		}

		toolInput := toolInputStr
		if arg1, ok := toolInputMap["__arg1"]; ok {
			toolInputCheck, ok := arg1.(string)
			if ok {
				toolInput = toolInputCheck
			}
		}

		contentMsg := "\n"
		if choice.Content != "" {
			contentMsg = fmt.Sprintf("responded: %s\n", choice.Content)
		}

		return []schema.AgentAction{
			{
				Tool:      functionName,
				ToolInput: toolInput,
				Log:       fmt.Sprintf("Invoking: %s with %s \n %s \n", functionName, toolInputStr, contentMsg),
				ToolID:    "",
			},
		}, nil, nil
	}

	return nil, &schema.AgentFinish{
		ReturnValues: map[string]any{
			"output": choice.Content,
		},
		Log: choice.Content,
	}, nil
}

func createOpenAIFunctionPromptWithSchema(
	systemMessage string,
	extraMessages []prompts.MessageFormatter,
) prompts.ChatPromptTemplate {
	messageFormatters := []prompts.MessageFormatter{prompts.NewSystemMessagePromptTemplate(systemMessage, nil)}
	messageFormatters = append(messageFormatters, extraMessages...)
	messageFormatters = append(messageFormatters, prompts.NewHumanMessagePromptTemplate("{{.input}}", []string{"input"}))
	messageFormatters = append(messageFormatters, prompts.MessagesPlaceholder{
		VariableName: agentScratchpad,
	})

	tmpl := prompts.NewChatPromptTemplate(messageFormatters)
	return tmpl
}
