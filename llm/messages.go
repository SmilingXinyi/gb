package llm

// Role is a strongly typed chat message role.
type Role string

const (
	// RoleSystem identifies a system instruction message.
	RoleSystem Role = "system"
	// RoleUser identifies an end-user message.
	RoleUser Role = "user"
	// RoleAssistant identifies an assistant reply.
	RoleAssistant Role = "assistant"
)

// Message is a business-layer message used to inject conversation history.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
	// Name optionally distinguishes multiple participants with the same role.
	Name string `json:"name,omitempty"`
}
