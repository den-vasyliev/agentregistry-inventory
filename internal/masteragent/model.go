package masteragent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"time"

	"google.golang.org/genai"

	"google.golang.org/adk/model"
)

// GatewayModel implements model.LLM for AgentGateway's OpenAI-compatible endpoint.
// It translates between ADK's genai types and OpenAI chat completion format,
// calling ModelCatalog.Spec.BaseURL with no API key (AgentGateway handles auth).
type GatewayModel struct {
	name    string // model identifier (e.g., "claude-sonnet-4-5-20250929")
	baseURL string // AgentGateway endpoint (e.g., "http://agentgateway:8080/openai")
	client  *http.Client
}

// NewGatewayModel creates a new GatewayModel.
// name is the model identifier from ModelCatalog.Spec.Model.
// baseURL is from ModelCatalog.Spec.BaseURL (e.g., "http://agentgateway:8080/openai").
func NewGatewayModel(name, baseURL string) *GatewayModel {
	return &GatewayModel{
		name:    name,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (m *GatewayModel) Name() string { return m.name }

// GenerateContent converts an ADK LLMRequest to OpenAI chat completion,
// calls AgentGateway, and converts the response back.
func (m *GatewayModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.doRequest(ctx, req)
		if err != nil {
			yield(nil, fmt.Errorf("gateway model request failed: %w", err))
			return
		}
		yield(resp, nil)
	}
}

func (m *GatewayModel) doRequest(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	// Convert ADK request → OpenAI format
	openAIReq := m.convertRequest(req)

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := m.baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	// Convert OpenAI response → ADK format
	var openAIResp openAIChatCompletionResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return m.convertResponse(&openAIResp), nil
}

// convertRequest converts ADK LLMRequest to OpenAI chat completion request
func (m *GatewayModel) convertRequest(req *model.LLMRequest) *openAIChatCompletionRequest {
	oaiReq := &openAIChatCompletionRequest{
		Model: m.name,
	}

	// Convert system instruction
	if req.Config != nil && req.Config.SystemInstruction != nil {
		for _, part := range req.Config.SystemInstruction.Parts {
			if part.Text != "" {
				oaiReq.Messages = append(oaiReq.Messages, openAIMessage{
					Role:    "system",
					Content: part.Text,
				})
			}
		}
	}

	// Convert contents to messages
	for _, content := range req.Contents {
		oaiReq.Messages = append(oaiReq.Messages, m.convertContentToMessages(content)...)
	}

	// Convert tools
	if req.Config != nil && req.Config.Tools != nil {
		for _, t := range req.Config.Tools {
			if t.FunctionDeclarations != nil {
				for _, fd := range t.FunctionDeclarations {
					oaiReq.Tools = append(oaiReq.Tools, openAITool{
						Type: "function",
						Function: openAIFunction{
							Name:        fd.Name,
							Description: fd.Description,
							Parameters:  schemaToMap(fd.Parameters),
						},
					})
				}
			}
		}
	}

	return oaiReq
}

// convertContentToMessages converts a genai.Content to one or more OpenAI messages.
// A single Content may produce multiple messages, e.g. an assistant message with
// tool_calls is followed by separate "tool" messages for each FunctionResponse.
func (m *GatewayModel) convertContentToMessages(content *genai.Content) []openAIMessage {
	if content == nil || len(content.Parts) == 0 {
		return nil
	}

	role := "user"
	if content.Role == "model" {
		role = "assistant"
	} else if content.Role != "" {
		role = string(content.Role)
	}

	var toolCalls []openAIToolCall
	var toolResponses []openAIMessage
	var textParts []string

	for _, part := range content.Parts {
		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			toolCallID := part.FunctionCall.ID
			if toolCallID == "" {
				toolCallID = part.FunctionCall.Name
			}
			toolCalls = append(toolCalls, openAIToolCall{
				ID:   toolCallID,
				Type: "function",
				Function: openAIFunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		} else if part.FunctionResponse != nil {
			respJSON, _ := json.Marshal(part.FunctionResponse.Response)
			toolCallID := part.FunctionResponse.ID
			if toolCallID == "" {
				toolCallID = part.FunctionResponse.Name
			}
			toolResponses = append(toolResponses, openAIMessage{
				Role:       "tool",
				Content:    string(respJSON),
				ToolCallID: toolCallID,
			})
		} else if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}

	var messages []openAIMessage

	if len(toolCalls) > 0 {
		msg := openAIMessage{
			Role:      "assistant",
			ToolCalls: toolCalls,
		}
		if len(textParts) > 0 {
			combined := ""
			for _, t := range textParts {
				combined += t
			}
			msg.Content = combined
		}
		messages = append(messages, msg)
		// Tool responses follow the assistant message
		messages = append(messages, toolResponses...)
		return messages
	}

	if len(toolResponses) > 0 {
		return toolResponses
	}

	combined := ""
	for _, t := range textParts {
		combined += t
	}
	return []openAIMessage{{
		Role:    role,
		Content: combined,
	}}
}

// convertResponse converts an OpenAI chat completion response to ADK LLMResponse
func (m *GatewayModel) convertResponse(resp *openAIChatCompletionResponse) *model.LLMResponse {
	llmResp := &model.LLMResponse{
		TurnComplete: true,
	}

	if len(resp.Choices) == 0 {
		llmResp.Content = genai.NewContentFromText("", "model")
		return llmResp
	}

	choice := resp.Choices[0]
	var parts []*genai.Part

	// Convert text content
	if choice.Message.Content != "" {
		parts = append(parts, genai.NewPartFromText(choice.Message.Content))
	}

	// Convert tool calls — store OpenAI tool call ID in FunctionCall.ID
	// so ADK preserves it through tool execution into FunctionResponse.ID
	for _, tc := range choice.Message.ToolCalls {
		var args map[string]any
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		part := genai.NewPartFromFunctionCall(tc.Function.Name, args)
		part.FunctionCall.ID = tc.ID
		parts = append(parts, part)
	}

	if len(parts) > 0 {
		llmResp.Content = genai.NewContentFromParts(parts, "model")
	} else {
		llmResp.Content = genai.NewContentFromText("", "model")
	}

	// Map finish reason
	switch choice.FinishReason {
	case "stop":
		llmResp.FinishReason = genai.FinishReasonStop
	case "tool_calls":
		llmResp.FinishReason = genai.FinishReasonStop
		llmResp.TurnComplete = false
	case "length":
		llmResp.FinishReason = genai.FinishReasonMaxTokens
	}

	// Map usage metadata
	if resp.Usage.TotalTokens > 0 {
		llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(resp.Usage.PromptTokens),
			CandidatesTokenCount: int32(resp.Usage.CompletionTokens),
			TotalTokenCount:      int32(resp.Usage.TotalTokens),
		}
	}

	return llmResp
}

// schemaToMap converts a genai.Schema to a map for JSON serialization
func schemaToMap(schema *genai.Schema) map[string]any {
	if schema == nil {
		return nil
	}
	// Marshal and unmarshal to get a map representation
	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var result map[string]any
	json.Unmarshal(data, &result)
	return result
}

// OpenAI API types

type openAIChatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatCompletionResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
