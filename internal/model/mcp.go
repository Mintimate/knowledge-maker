package model

// ==================== MCP 工具相关 ====================

// MCPTool MCP 工具定义
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// MCPToolCallRequest MCP 工具调用请求
type MCPToolCallRequest struct {
	ToolName  string                 `json:"tool_name" binding:"required"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPToolCallResponse MCP 工具调用响应
type MCPToolCallResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Message string      `json:"message,omitempty"`
}

// MCPToolsListResponse MCP 工具列表响应
type MCPToolsListResponse struct {
	Success bool      `json:"success"`
	Tools   []MCPTool `json:"tools"`
}

// ==================== LLM 聊天相关 ====================

// LLMChatMessage LLM 聊天消息
type LLMChatMessage struct {
	Role       string        `json:"role"` // system, user, assistant, tool
	Content    string        `json:"content,omitempty"`
	ToolCalls  []LLMToolCall `json:"tool_calls,omitempty"`   // assistant 消息中的工具调用
	ToolCallID string        `json:"tool_call_id,omitempty"` // tool 消息中的工具调用ID
	Name       string        `json:"name,omitempty"`         // tool 消息中的工具名称
}

// LLMToolCall LLM 工具调用
type LLMToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"` // "function"
	Function LLMFunctionCall `json:"function"`
}

// LLMFunctionCall 函数调用详情
type LLMFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON 字符串
}

// LLMToolDef LLM 工具定义（OpenAI function calling 格式）
type LLMToolDef struct {
	Type     string         `json:"type"` // "function"
	Function LLMFunctionDef `json:"function"`
}

// LLMFunctionDef 函数定义
type LLMFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// LLMChatRequest LLM 聊天请求
type LLMChatRequest struct {
	Messages []LLMChatMessage `json:"messages" binding:"required"`
	Tools    []LLMToolDef     `json:"tools,omitempty"` // 可用工具列表
	Stream   bool             `json:"stream"`          // 是否流式输出
}

// LLMChatResponse LLM 非流式聊天响应
type LLMChatResponse struct {
	Success bool            `json:"success"`
	Message *LLMChatMessage `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// LLMStreamChunk LLM 流式响应块
type LLMStreamChunk struct {
	Delta        *LLMChatMessageDelta `json:"delta,omitempty"`
	FinishReason string               `json:"finish_reason,omitempty"`
}

// LLMChatMessageDelta 流式消息增量
type LLMChatMessageDelta struct {
	Role             string             `json:"role,omitempty"`
	Content          string             `json:"content,omitempty"`
	ReasoningContent string             `json:"reasoning_content,omitempty"`
	ToolCalls        []LLMToolCallDelta `json:"tool_calls,omitempty"`
}

// LLMToolCallDelta 流式工具调用增量
type LLMToolCallDelta struct {
	Index    int                   `json:"index"`
	ID       string                `json:"id,omitempty"`
	Type     string                `json:"type,omitempty"`
	Function *LLMFunctionCallDelta `json:"function,omitempty"`
}

// LLMFunctionCallDelta 流式函数调用增量
type LLMFunctionCallDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
