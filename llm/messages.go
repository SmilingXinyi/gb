package llm

// Role represents a chat message role in the OpenAI chat completions API.
type Role string

const (
	// RoleSystem identifies a system instruction message.
	RoleSystem Role = "system"
	// RoleUser identifies an end-user message.
	RoleUser Role = "user"
	// RoleAssistant identifies an assistant reply.
	RoleAssistant Role = "assistant"
)

// Message is a single chat completion message.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// SystemMessage builds a system role message.
func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// UserMessage builds a user role message.
func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// AssistantMessage builds an assistant role message.
func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}
