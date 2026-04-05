package provider

const (
	// RoleSystem 标识系统消息。
	RoleSystem = "system"
	// RoleUser 标识用户消息。
	RoleUser = "user"
	// RoleAssistant 标识助手消息。
	RoleAssistant = "assistant"
	// RoleTool 标识工具结果消息。
	RoleTool = "tool"
)

// Message 表示对话中的单条消息。
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	IsError    bool       `json:"is_error,omitempty"`
}

// ToolCall 表示模型发起的工具调用请求。
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolSpec 表示暴露给模型的可调用工具描述。
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
}

// ChatRequest 是 provider.Chat() 的请求参数。
type ChatRequest struct {
	Model        string     `json:"model"`
	SystemPrompt string     `json:"system_prompt"`
	Messages     []Message  `json:"messages"`
	Tools        []ToolSpec `json:"tools,omitempty"`
}

// Usage 记录本次请求的 token 使用统计。
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// StreamEventType 定义流式事件类型。
type StreamEventType string

const (
	// StreamEventTextDelta 表示模型输出的文本片段。
	StreamEventTextDelta StreamEventType = "text_delta"
	// StreamEventToolCallStart 表示模型开始请求工具调用。
	StreamEventToolCallStart StreamEventType = "tool_call_start"
	// StreamEventToolCallDelta 表示工具调用参数的增量片段。
	StreamEventToolCallDelta StreamEventType = "tool_call_delta"
	// StreamEventMessageDone 表示本轮消息完成，并携带最终统计信息。
	StreamEventMessageDone StreamEventType = "message_done"
)

// StreamEvent 表示 provider 向 runtime 推送的流式事件。
type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Payload interface{}     `json:"payload,omitempty"`
}

// TextDeltaPayload 表示文本增量事件的载荷。
type TextDeltaPayload struct {
	Text string `json:"text"`
}

// ToolCallStartPayload 表示工具调用开始事件的载荷。
type ToolCallStartPayload struct {
	Index int    `json:"index"`
	ID    string `json:"id"`
	Name  string `json:"name"`
}

// ToolCallDeltaPayload 表示工具调用参数增量事件的载荷。
type ToolCallDeltaPayload struct {
	Index          int    `json:"index"`
	ID             string `json:"id"`
	ArgumentsDelta string `json:"arguments_delta"`
}

// MessageDonePayload 表示消息完成事件的载荷。
type MessageDonePayload struct {
	FinishReason string `json:"finish_reason"`
	Usage        *Usage `json:"usage"`
}

// NewTextDeltaStreamEvent 创建文本增量流事件。
func NewTextDeltaStreamEvent(text string) StreamEvent {
	return StreamEvent{
		Type:    StreamEventTextDelta,
		Payload: TextDeltaPayload{Text: text},
	}
}

// NewToolCallStartStreamEvent 创建工具调用开始流事件。
func NewToolCallStartStreamEvent(index int, id, name string) StreamEvent {
	return StreamEvent{
		Type:    StreamEventToolCallStart,
		Payload: ToolCallStartPayload{Index: index, ID: id, Name: name},
	}
}

// NewToolCallDeltaStreamEvent 创建工具调用参数增量流事件。
func NewToolCallDeltaStreamEvent(index int, id, argumentsDelta string) StreamEvent {
	return StreamEvent{
		Type:    StreamEventToolCallDelta,
		Payload: ToolCallDeltaPayload{Index: index, ID: id, ArgumentsDelta: argumentsDelta},
	}
}

// NewMessageDoneStreamEvent 创建消息完成流事件。
func NewMessageDoneStreamEvent(finishReason string, usage *Usage) StreamEvent {
	return StreamEvent{
		Type:    StreamEventMessageDone,
		Payload: MessageDonePayload{FinishReason: finishReason, Usage: usage},
	}
}
