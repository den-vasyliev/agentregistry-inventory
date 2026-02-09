package masteragent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/genai"

	"google.golang.org/adk/model"
)

func TestGatewayModel_Name(t *testing.T) {
	m := NewGatewayModel("claude-sonnet-4-5-20250929", "http://localhost:8080", "")
	assert.Equal(t, "claude-sonnet-4-5-20250929", m.Name())
}

func TestGatewayModel_GenerateContent_TextResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		resp := openAIChatCompletionResponse{
			Choices: []openAIChoice{
				{
					Message:      openAIMessage{Role: "assistant", Content: "Hello from the LLM"},
					FinishReason: "stop",
				},
			},
			Usage: openAIUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	m := NewGatewayModel("test-model", server.URL, "")
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("Hello", "user"),
		},
	}

	var got *model.LLMResponse
	var gotErr error
	for resp, err := range m.GenerateContent(context.Background(), req, false) {
		got = resp
		gotErr = err
	}

	require.NoError(t, gotErr)
	require.NotNil(t, got)
	assert.True(t, got.TurnComplete)
	assert.Equal(t, genai.FinishReasonStop, got.FinishReason)

	require.NotNil(t, got.Content)
	require.Len(t, got.Content.Parts, 1)
	assert.Equal(t, "Hello from the LLM", got.Content.Parts[0].Text)

	require.NotNil(t, got.UsageMetadata)
	assert.Equal(t, int32(10), got.UsageMetadata.PromptTokenCount)
	assert.Equal(t, int32(5), got.UsageMetadata.CandidatesTokenCount)
	assert.Equal(t, int32(15), got.UsageMetadata.TotalTokenCount)
}

func TestGatewayModel_GenerateContent_ToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIChatCompletionResponse{
			Choices: []openAIChoice{
				{
					Message: openAIMessage{
						Role: "assistant",
						ToolCalls: []openAIToolCall{
							{
								ID:   "call_1",
								Type: "function",
								Function: openAIFunctionCall{
									Name:      "get_world_state",
									Arguments: `{"foo":"bar"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	m := NewGatewayModel("test-model", server.URL, "")
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("check state", "user"),
		},
	}

	var got *model.LLMResponse
	for resp, err := range m.GenerateContent(context.Background(), req, false) {
		require.NoError(t, err)
		got = resp
	}

	require.NotNil(t, got)
	assert.False(t, got.TurnComplete, "tool_calls should set TurnComplete=false")

	require.NotNil(t, got.Content)
	require.Len(t, got.Content.Parts, 1)

	fc := got.Content.Parts[0].FunctionCall
	require.NotNil(t, fc)
	assert.Equal(t, "get_world_state", fc.Name)
	assert.Equal(t, "bar", fc.Args["foo"])
	assert.Equal(t, "call_1", fc.ID, "tool call ID should be preserved in FunctionCall.ID")
}

func TestGatewayModel_GenerateContent_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	m := NewGatewayModel("test-model", server.URL, "")
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("hello", "user"),
		},
	}

	for _, err := range m.GenerateContent(context.Background(), req, false) {
		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")
		return
	}
	t.Fatal("expected at least one iteration")
}

func TestGatewayModel_GenerateContent_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIChatCompletionResponse{Choices: []openAIChoice{}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	m := NewGatewayModel("test-model", server.URL, "")
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("hello", "user"),
		},
	}

	var got *model.LLMResponse
	for resp, err := range m.GenerateContent(context.Background(), req, false) {
		require.NoError(t, err)
		got = resp
	}

	require.NotNil(t, got)
	require.NotNil(t, got.Content)
	assert.Equal(t, "", got.Content.Parts[0].Text)
}

func TestGatewayModel_ConvertRequest_SystemInstruction(t *testing.T) {
	m := NewGatewayModel("test-model", "http://unused", "")

	req := &model.LLMRequest{
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText("You are helpful", ""),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText("Hello", "user"),
		},
	}

	oaiReq := m.convertRequest(req)
	require.Len(t, oaiReq.Messages, 2)
	assert.Equal(t, "system", oaiReq.Messages[0].Role)
	assert.Equal(t, "You are helpful", oaiReq.Messages[0].Content)
	assert.Equal(t, "user", oaiReq.Messages[1].Role)
}

func TestGatewayModel_ConvertRequest_Tools(t *testing.T) {
	m := NewGatewayModel("test-model", "http://unused", "")

	req := &model.LLMRequest{
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{
				{
					FunctionDeclarations: []*genai.FunctionDeclaration{
						{
							Name:        "get_state",
							Description: "Get the current state",
							Parameters: &genai.Schema{
								Type: genai.TypeObject,
								Properties: map[string]*genai.Schema{
									"key": {Type: genai.TypeString},
								},
							},
						},
					},
				},
			},
		},
		Contents: []*genai.Content{
			genai.NewContentFromText("hi", "user"),
		},
	}

	oaiReq := m.convertRequest(req)
	require.Len(t, oaiReq.Tools, 1)
	assert.Equal(t, "function", oaiReq.Tools[0].Type)
	assert.Equal(t, "get_state", oaiReq.Tools[0].Function.Name)
	assert.Equal(t, "Get the current state", oaiReq.Tools[0].Function.Description)
	require.NotNil(t, oaiReq.Tools[0].Function.Parameters)
}

func TestGatewayModel_ConvertRequest_FunctionCallContent(t *testing.T) {
	m := NewGatewayModel("test-model", "http://unused", "")

	part := genai.NewPartFromFunctionCall("get_state", map[string]any{"key": "val"})
	part.FunctionCall.ID = "call_abc123"

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromParts([]*genai.Part{part}, "model"),
		},
	}

	oaiReq := m.convertRequest(req)
	require.Len(t, oaiReq.Messages, 1)
	msg := oaiReq.Messages[0]
	assert.Equal(t, "assistant", msg.Role)
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "get_state", msg.ToolCalls[0].Function.Name)
	assert.Equal(t, "call_abc123", msg.ToolCalls[0].ID, "tool call ID from FunctionCall.ID")
}

func TestGatewayModel_ConvertRequest_FunctionResponseContent(t *testing.T) {
	m := NewGatewayModel("test-model", "http://unused", "")

	part := genai.NewPartFromFunctionResponse("get_state", map[string]any{"status": "ok"})
	part.FunctionResponse.ID = "call_abc123"

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromParts([]*genai.Part{part}, "user"),
		},
	}

	oaiReq := m.convertRequest(req)
	require.Len(t, oaiReq.Messages, 1)
	msg := oaiReq.Messages[0]
	assert.Equal(t, "tool", msg.Role)
	assert.Equal(t, "call_abc123", msg.ToolCallID, "tool call ID from FunctionResponse.ID")
	assert.Contains(t, msg.Content, "ok")
}

func TestGatewayModel_ConvertResponse_FinishReasons(t *testing.T) {
	m := NewGatewayModel("test-model", "http://unused", "")

	tests := []struct {
		name         string
		finishReason string
		wantFinish   genai.FinishReason
		wantComplete bool
	}{
		{"stop", "stop", genai.FinishReasonStop, true},
		{"tool_calls", "tool_calls", genai.FinishReasonStop, false},
		{"length", "length", genai.FinishReasonMaxTokens, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &openAIChatCompletionResponse{
				Choices: []openAIChoice{
					{
						Message:      openAIMessage{Role: "assistant", Content: "text"},
						FinishReason: tt.finishReason,
					},
				},
			}

			got := m.convertResponse(resp)
			assert.Equal(t, tt.wantFinish, got.FinishReason)
			assert.Equal(t, tt.wantComplete, got.TurnComplete)
		})
	}
}

func TestGatewayModel_ConvertResponse_Usage(t *testing.T) {
	m := NewGatewayModel("test-model", "http://unused", "")

	resp := &openAIChatCompletionResponse{
		Choices: []openAIChoice{
			{Message: openAIMessage{Content: "hi"}, FinishReason: "stop"},
		},
		Usage: openAIUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
	}

	got := m.convertResponse(resp)
	require.NotNil(t, got.UsageMetadata)
	assert.Equal(t, int32(100), got.UsageMetadata.PromptTokenCount)
	assert.Equal(t, int32(50), got.UsageMetadata.CandidatesTokenCount)
	assert.Equal(t, int32(150), got.UsageMetadata.TotalTokenCount)
}

func TestGatewayModel_ToolCallIDRoundTrip(t *testing.T) {
	// Simulates: LLM returns tool_calls → ADK executes tools → history sent back to LLM
	// This is the exact scenario that was failing with OpenAI's error about missing tool_call_id responses.
	m := NewGatewayModel("test-model", "http://unused", "")

	// Step 1: LLM returns a response with tool calls
	resp := &openAIChatCompletionResponse{
		Choices: []openAIChoice{
			{
				Message: openAIMessage{
					Role: "assistant",
					ToolCalls: []openAIToolCall{
						{ID: "call_AfjDRqMf", Type: "function", Function: openAIFunctionCall{Name: "get_world_state", Arguments: `{}`}},
						{ID: "call_BxjKLm99", Type: "function", Function: openAIFunctionCall{Name: "list_catalog", Arguments: `{"type":"servers"}`}},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}

	llmResp := m.convertResponse(resp)
	require.Len(t, llmResp.Content.Parts, 2)
	assert.Equal(t, "call_AfjDRqMf", llmResp.Content.Parts[0].FunctionCall.ID)
	assert.Equal(t, "call_BxjKLm99", llmResp.Content.Parts[1].FunctionCall.ID)

	// Step 2: ADK executes tools and creates FunctionResponse content (copies IDs)
	toolResponseContent := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{FunctionResponse: &genai.FunctionResponse{ID: "call_AfjDRqMf", Name: "get_world_state", Response: map[string]any{"summary": "all good"}}},
			{FunctionResponse: &genai.FunctionResponse{ID: "call_BxjKLm99", Name: "list_catalog", Response: map[string]any{"count": 5}}},
		},
	}

	// Step 3: Build the full conversation history as ADK would send it
	req := &model.LLMRequest{
		Contents: []*genai.Content{
			genai.NewContentFromText("check state", "user"),
			llmResp.Content,     // assistant message with tool_calls
			toolResponseContent, // tool responses
		},
	}

	oaiReq := m.convertRequest(req)

	// Should produce: user message, assistant with tool_calls, tool response 1, tool response 2
	require.Len(t, oaiReq.Messages, 4)

	assert.Equal(t, "user", oaiReq.Messages[0].Role)

	assert.Equal(t, "assistant", oaiReq.Messages[1].Role)
	require.Len(t, oaiReq.Messages[1].ToolCalls, 2)
	assert.Equal(t, "call_AfjDRqMf", oaiReq.Messages[1].ToolCalls[0].ID)
	assert.Equal(t, "call_BxjKLm99", oaiReq.Messages[1].ToolCalls[1].ID)

	assert.Equal(t, "tool", oaiReq.Messages[2].Role)
	assert.Equal(t, "call_AfjDRqMf", oaiReq.Messages[2].ToolCallID)

	assert.Equal(t, "tool", oaiReq.Messages[3].Role)
	assert.Equal(t, "call_BxjKLm99", oaiReq.Messages[3].ToolCallID)
}

func TestGatewayModel_ConvertRequest_MultipleFunctionResponses(t *testing.T) {
	// A single Content with multiple FunctionResponse parts should produce
	// multiple "tool" messages (one per response).
	m := NewGatewayModel("test-model", "http://unused", "")

	req := &model.LLMRequest{
		Contents: []*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{FunctionResponse: &genai.FunctionResponse{ID: "call_1", Name: "tool_a", Response: map[string]any{"a": 1}}},
					{FunctionResponse: &genai.FunctionResponse{ID: "call_2", Name: "tool_b", Response: map[string]any{"b": 2}}},
					{FunctionResponse: &genai.FunctionResponse{ID: "call_3", Name: "tool_c", Response: map[string]any{"c": 3}}},
				},
			},
		},
	}

	oaiReq := m.convertRequest(req)
	require.Len(t, oaiReq.Messages, 3)

	for i, expected := range []struct {
		toolCallID string
	}{
		{"call_1"},
		{"call_2"},
		{"call_3"},
	} {
		assert.Equal(t, "tool", oaiReq.Messages[i].Role)
		assert.Equal(t, expected.toolCallID, oaiReq.Messages[i].ToolCallID)
	}
}

func TestGatewayModel_ConvertResponse_NoUsage(t *testing.T) {
	m := NewGatewayModel("test-model", "http://unused", "")

	resp := &openAIChatCompletionResponse{
		Choices: []openAIChoice{
			{Message: openAIMessage{Content: "hi"}, FinishReason: "stop"},
		},
		Usage: openAIUsage{}, // zero values
	}

	got := m.convertResponse(resp)
	assert.Nil(t, got.UsageMetadata)
}
