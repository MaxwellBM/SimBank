package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	bankmcp "banca-backend/internal/mcp"
)

// ──────────────────────────────────────────────
// OpenRouter API types
// ──────────────────────────────────────────────

type ORMessage struct {
	Role       string      `json:"role"`
	Content    *string     `json:"content,omitempty"`
	ToolCalls  []ORToolCall `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type ORToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function ORToolCallFunction `json:"function"`
}

type ORToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ORToolDefinition struct {
	Type     string         `json:"type"`
	Function ORFunctionSpec `json:"function"`
}

type ORFunctionSpec struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  any         `json:"parameters"`
}

type ORRequest struct {
	Model       string             `json:"model"`
	Messages    []ORMessage        `json:"messages"`
	Tools       []ORToolDefinition `json:"tools,omitempty"`
	ToolChoice  any                `json:"tool_choice,omitempty"`
}

type ORResponse struct {
	ID      string        `json:"id"`
	Choices []ORChoice    `json:"choices"`
	Error   *ORAPIError   `json:"error,omitempty"`
}

type ORChoice struct {
	Index   int        `json:"index"`
	Message ORMessage  `json:"message"`
}

type ORAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ──────────────────────────────────────────────
// ChatHandler
// ──────────────────────────────────────────────

type ChatHandler struct {
	mcpFactory     *bankmcp.ServerFactory
	openRouterKey  string
	model          string
	openRouterURL  string
	httpClient     *http.Client
}

func NewChatHandler(factory *bankmcp.ServerFactory, apiKey, model string) *ChatHandler {
	return &ChatHandler{
		mcpFactory:    factory,
		openRouterKey: apiKey,
		model:         model,
		openRouterURL: "https://openrouter.ai/api/v1/chat/completions",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (h *ChatHandler) IsConfigured() bool {
	return h.openRouterKey != ""
}

// ToolDefinitionsForUser returns the MCP tool definitions formatted for OpenRouter.
func (h *ChatHandler) ToolDefinitionsForUser(userID string) ([]ORToolDefinition, error) {
	server := h.mcpFactory.NewServer(userID)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		if err := server.Run(ctx, serverTransport); err != nil {
			log.Printf("MCP server exited: %v", err)
		}
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "chat-server", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect mcp: %w", err)
	}
	defer session.Close()

	listResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	tools := make([]ORToolDefinition, 0, len(listResult.Tools))
	for _, t := range listResult.Tools {
		spec := ORFunctionSpec{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		}
		tools = append(tools, ORToolDefinition{
			Type:     "function",
			Function: spec,
		})
	}
	return tools, nil
}

// ProcessMessage handles a single chat turn: sends the message + history to
// OpenRouter, processes any tool calls, and returns the final response.
func (h *ChatHandler) ProcessMessage(ctx context.Context, userID string, message string, history []ORMessage) (
	replyText string, pendingAction *PendingAction, requiresConfirmation bool, finalErr error,
) {
	if !h.IsConfigured() {
		return "", nil, false, errors.New("Chat IA no configurado. Agregue OPENROUTER_API_KEY a su archivo .env")
	}

	orHistory := buildHistory(message, history)

	tools, err := h.ToolDefinitionsForUser(userID)
	if err != nil {
		return "", nil, false, fmt.Errorf("error al preparar herramientas: %w", err)
	}

	orResp, err := h.callOpenRouter(ctx, orHistory, tools)
	if err != nil {
		return "", nil, false, fmt.Errorf("error al comunicarse con la IA: %w", err)
	}

	if orResp.Error != nil {
		return "", nil, false, fmt.Errorf("error de la IA: %s", orResp.Error.Message)
	}

	if len(orResp.Choices) == 0 {
		return "", nil, false, errors.New("la IA no generó una respuesta")
	}

	msg := orResp.Choices[0].Message

	if len(msg.ToolCalls) == 0 {
		if msg.Content != nil {
			return *msg.Content, nil, false, nil
		}
		return "", nil, false, nil
	}

	// ── Handle tool calls ──
	tc := msg.ToolCalls[0]

	if tc.Function.Name == "transfer" {
		// Transfer is sensitive: return pending action for user confirmation.
		action, err := parseTransferAction(userID, tc.Function.Arguments)
		if err != nil {
			return "", nil, false, fmt.Errorf("error al interpretar transferencia: %w", err)
		}
		return action.Summary, action, true, nil
	}

	// Execute non-sensitive tools via MCP.
	server := h.mcpFactory.NewServer(userID)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go func() {
		if err := server.Run(ctx, serverTransport); err != nil {
			log.Printf("MCP server exited: %v", err)
		}
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "chat-executor", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", nil, false, fmt.Errorf("error al conectar con el sistema bancario: %w", err)
	}
	defer session.Close()

	var args map[string]any
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return "", nil, false, fmt.Errorf("error al interpretar argumentos: %w", err)
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tc.Function.Name,
		Arguments: args,
	})
	if err != nil {
		return "", nil, false, fmt.Errorf("error al ejecutar %s: %w", tc.Function.Name, err)
	}

	toolResultText := ""
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			toolResultText = tc.Text
			break
		}
	}
	if toolResultText == "" {
		toolResultText = "Operación completada."
	}

	// Add tool result to the conversation and call OpenRouter again.
	orHistory = append(orHistory, msg)
	orHistory = append(orHistory, ORMessage{
		Role:       "tool",
		ToolCallID: tc.ID,
		Content:    &toolResultText,
	})

	brainResp, err := h.callOpenRouter(ctx, orHistory, tools)
	if err != nil {
		return toolResultText, nil, false, nil
	}
	if brainResp.Error != nil {
		return toolResultText, nil, false, nil
	}

	if len(brainResp.Choices) > 0 && brainResp.Choices[0].Message.Content != nil {
		return *brainResp.Choices[0].Message.Content, nil, false, nil
	}

	return toolResultText, nil, false, nil
}

// ExecuteConfirmedTransfer executes a transfer that was previously confirmed.
// Uses ServerFactory.ExecuteTransfer directly (bypasses the MCP "transfer"
// tool, which is designed to only prepare confirmation requests).
func (h *ChatHandler) ExecuteConfirmedTransfer(ctx context.Context, userID, toAccount string, amount float64) (string, error) {
	return h.mcpFactory.ExecuteTransfer(ctx, userID, toAccount, amount)
}

// ──────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────

func buildHistory(message string, history []ORMessage) []ORMessage {
	system := ORMessage{
		Role: "system",
		Content: ptrString(`Eres un asistente bancario amable y profesional. 
Siempre respondes en español, de forma clara y concisa.
Usas las herramientas disponibles para consultar datos reales del banco — NUNCA inventes cifras ni estados de cuenta.
Cuando te pidan una operación como depósito o retiro, usa la herramienta correspondiente.
Para transferencias, la herramienta 'transfer' te devolverá un resumen — indícaselo al usuario y pídele confirmación.`),
	}

	result := []ORMessage{system}
	result = append(result, history...)
	result = append(result, ORMessage{
		Role:    "user",
		Content: ptrString(message),
	})
	return result
}

func (h *ChatHandler) callOpenRouter(ctx context.Context, messages []ORMessage, tools []ORToolDefinition) (*ORResponse, error) {
	body := ORRequest{
		Model:    h.model,
		Messages: messages,
	}
	if len(tools) > 0 {
		body.Tools = tools
		body.ToolChoice = "auto"
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.openRouterURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.openRouterKey)
	req.Header.Set("HTTP-Referer", "https://simbank.app")
	req.Header.Set("X-Title", "SimBank")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var orResp ORResponse
	if err := json.Unmarshal(raw, &orResp); err != nil {
		return nil, fmt.Errorf("parse response: %w (body: %s)", err, string(raw))
	}

	return &orResp, nil
}

// PendingAction holds the details of a transfer awaiting confirmation.
type PendingAction struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	FromAccount string  `json:"from_account"`
	ToAccount   string  `json:"to_account"`
	Amount      float64 `json:"amount"`
	RecipientName string `json:"recipient_name"`
	Summary     string  `json:"summary"`
}

func parseTransferAction(userID string, argumentsJSON string) (*PendingAction, error) {
	var args struct {
		ToAccount string  `json:"to_account_number"`
		Amount    float64 `json:"amount"`
	}
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	return &PendingAction{
		ID:        fmt.Sprintf("transfer_%d", time.Now().UnixNano()),
		Type:      "transfer",
		FromAccount: userID,
		ToAccount: args.ToAccount,
		Amount:    args.Amount,
		Summary: fmt.Sprintf(
			"Vas a transferir L %.2f a la cuenta %s. ¿Confirmas?",
			args.Amount, args.ToAccount,
		),
	}, nil
}

func ptrString(s string) *string {
	return &s
}
