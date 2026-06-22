// Package mcp implements a Model Context Protocol server that exposes
// banking operations as tools. Each tool operates on the authenticated user's
// account (userID is injected via closure when the server is created).
//
// The MCP server is designed to be created per-request (or per-session) with
// the userID captured in each tool handler's closure. This avoids relying on
// any MCP-native "user" concept, which doesn't exist in the protocol.
//
// Architecture decision:
//
//	We create the MCP server per-request via ServerFactory.NewServer(userID)
//	and connect to it via in-memory transport. This is clean, testable, and
//	avoids duplicating business logic — the MCP handlers call the same
//	internal/ledger and internal/db functions as the HTTP handlers.
package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"banca-backend/internal/db"
	"banca-backend/internal/ledger"
)

// ──────────────────────────────────────────────
// ServerFactory creates MCP Server instances scoped to a userID.
// ──────────────────────────────────────────────

type ServerFactory struct {
	store  *db.PostgresStore
	ledger *ledger.Ledger
}

func NewServerFactory(store *db.PostgresStore, l *ledger.Ledger) *ServerFactory {
	return &ServerFactory{store: store, ledger: l}
}

func (f *ServerFactory) NewServer(userID string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "simbank-mcp", Version: "v1.0.0"}, nil)

	f.registerGetBalance(server, userID)
	f.registerGetAccountInfo(server, userID)
	f.registerDeposit(server, userID)
	f.registerWithdraw(server, userID)
	f.registerTransfer(server, userID)
	f.registerGetTransactionHistory(server, userID)

	return server
}

// ──────────────────────────────────────────────
// 1. get_balance
// ──────────────────────────────────────────────

type GetBalanceInput struct{}

type GetBalanceOutput struct {
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

func (f *ServerFactory) registerGetBalance(server *mcp.Server, userID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_balance",
		Description: "Obtiene el saldo actual de la cuenta del usuario autenticado.",
		Title:       "Consultar saldo",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ GetBalanceInput) (*mcp.CallToolResult, GetBalanceOutput, error) {
		u, err := f.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, GetBalanceOutput{}, fmt.Errorf("usuario no encontrado: %w", err)
		}

		balance, err := f.ledger.GetBalance(u.TigerBeetleAccountID)
		if err != nil {
			return nil, GetBalanceOutput{}, fmt.Errorf("error al consultar saldo: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Su saldo actual es de L %.2f", balance)},
			},
		}, GetBalanceOutput{Balance: balance, Currency: "HNL"}, nil
	})
}

// ──────────────────────────────────────────────
// 2. get_account_info
// ──────────────────────────────────────────────

type GetAccountInfoInput struct{}

type GetAccountInfoOutput struct {
	AccountNumber string  `json:"account_number"`
	FullName      string  `json:"full_name"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
}

func (f *ServerFactory) registerGetAccountInfo(server *mcp.Server, userID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_account_info",
		Description: "Obtiene información detallada de la cuenta: número de cuenta, nombre del titular y saldo actual.",
		Title:       "Información de cuenta",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ GetAccountInfoInput) (*mcp.CallToolResult, GetAccountInfoOutput, error) {
		u, err := f.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, GetAccountInfoOutput{}, fmt.Errorf("usuario no encontrado: %w", err)
		}

		balance, err := f.ledger.GetBalance(u.TigerBeetleAccountID)
		if err != nil {
			return nil, GetAccountInfoOutput{}, fmt.Errorf("error al consultar saldo: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Cuenta: %s\nTitular: %s\nSaldo: L %.2f", u.AccountNumber, u.FullName, balance)},
			},
		}, GetAccountInfoOutput{
			AccountNumber: u.AccountNumber,
			FullName:      u.FullName,
			Balance:       balance,
			Currency:      "HNL",
		}, nil
	})
}

// ──────────────────────────────────────────────
// 3. deposit
// ──────────────────────────────────────────────

type DepositInput struct {
	Amount float64 `json:"amount" jsonschema:"Monto a depositar en dólares (ej: 50.00)"`
}

type DepositOutput struct {
	NewBalance float64 `json:"new_balance"`
	Message    string  `json:"message"`
}

func (f *ServerFactory) registerDeposit(server *mcp.Server, userID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "deposit",
		Description: "Realiza un depósito a la cuenta del usuario autenticado.",
		Title:       "Depositar",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in DepositInput) (*mcp.CallToolResult, DepositOutput, error) {
		if in.Amount <= 0 {
			return nil, DepositOutput{}, fmt.Errorf("el monto debe ser positivo")
		}

		u, err := f.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, DepositOutput{}, fmt.Errorf("usuario no encontrado: %w", err)
		}

		if err := f.ledger.Deposit(u.TigerBeetleAccountID, in.Amount); err != nil {
			return nil, DepositOutput{}, fmt.Errorf("error al depositar: %w", err)
		}

		balance, err := f.ledger.GetBalance(u.TigerBeetleAccountID)
		if err != nil {
			return nil, DepositOutput{}, fmt.Errorf("depósito exitoso pero no se pudo consultar el nuevo saldo: %w", err)
		}

		msg := fmt.Sprintf("Depósito exitoso. Se depositaron L %.2f. Su nuevo saldo es L %.2f.", in.Amount, balance)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, DepositOutput{NewBalance: balance, Message: msg}, nil
	})
}

// ──────────────────────────────────────────────
// 4. withdraw
// ──────────────────────────────────────────────

type WithdrawInput struct {
	Amount float64 `json:"amount" jsonschema:"Monto a retirar en dólares (ej: 20.00)"`
}

type WithdrawOutput struct {
	NewBalance float64 `json:"new_balance"`
	Message    string  `json:"message"`
}

func (f *ServerFactory) registerWithdraw(server *mcp.Server, userID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "withdraw",
		Description: "Realiza un retiro de la cuenta del usuario autenticado. Puede fallar si no hay fondos suficientes.",
		Title:       "Retirar",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in WithdrawInput) (*mcp.CallToolResult, WithdrawOutput, error) {
		if in.Amount <= 0 {
			return nil, WithdrawOutput{}, fmt.Errorf("el monto debe ser positivo")
		}

		u, err := f.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, WithdrawOutput{}, fmt.Errorf("usuario no encontrado: %w", err)
		}

		if err := f.ledger.Withdraw(u.TigerBeetleAccountID, in.Amount); err != nil {
			return nil, WithdrawOutput{}, fmt.Errorf("error al retirar: %w", err)
		}

		balance, err := f.ledger.GetBalance(u.TigerBeetleAccountID)
		if err != nil {
			return nil, WithdrawOutput{}, fmt.Errorf("retiro exitoso pero no se pudo consultar el nuevo saldo: %w", err)
		}

		msg := fmt.Sprintf("Retiro exitoso. Se retiraron L %.2f. Su nuevo saldo es L %.2f.", in.Amount, balance)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		}, WithdrawOutput{NewBalance: balance, Message: msg}, nil
	})
}

// ──────────────────────────────────────────────
// 5. transfer  —  SENSITIVE: does NOT execute the transfer
//
// This tool is intentionally designed NOT to execute the transfer directly.
// Instead, it looks up the destination account, validates the input, and
// returns a structured response that indicates the action requires
// confirmation. The actual execution happens only when the user explicitly
// confirms via the /api/chat/confirm endpoint (see FASE 9).
//
// Rationale (also required by the test specification):
//
//	"La IA debe confirmar acciones críticas antes de ejecutarlas"
//
// Transfers are irreversible in a banking context. Allowing an LLM to
// execute transfers without explicit user confirmation would violate
// basic safety and UX principles. The two-step flow (propose → confirm)
// ensures the user always has the final say on money movement.
// ──────────────────────────────────────────────

type TransferInput struct {
	ToAccount   string  `json:"to_account_number" jsonschema:"Número de cuenta destino (ej: 001-0002)"`
	Amount      float64 `json:"amount" jsonschema:"Monto a transferir en dólares"`
}

type TransferRecipientInfo struct {
	AccountNumber string `json:"account_number"`
	FullName      string `json:"full_name"`
}

type TransferOutput struct {
	RequiresConfirmation bool                 `json:"requires_confirmation"`
	Amount               float64              `json:"amount"`
	Recipient            TransferRecipientInfo `json:"recipient"`
	Summary              string               `json:"summary"`
}

func (f *ServerFactory) registerTransfer(server *mcp.Server, userID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "transfer",
		Description: "Prepara una transferencia a otra cuenta. NO ejecuta la transferencia automáticamente — requiere confirmación explícita del usuario antes de ejecutarse.",
		Title:       "Transferir (requiere confirmación)",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in TransferInput) (*mcp.CallToolResult, TransferOutput, error) {
		if in.Amount <= 0 {
			return nil, TransferOutput{}, fmt.Errorf("el monto debe ser positivo")
		}
		if in.ToAccount == "" {
			return nil, TransferOutput{}, fmt.Errorf("debe especificar una cuenta destino")
		}

		u, err := f.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, TransferOutput{}, fmt.Errorf("usuario no encontrado: %w", err)
		}

		if in.ToAccount == u.AccountNumber {
			return nil, TransferOutput{}, fmt.Errorf("no puede transferirse dinero a sí mismo")
		}

		dest, err := f.store.GetUserByAccountNumber(ctx, in.ToAccount)
		if err != nil {
			return nil, TransferOutput{}, fmt.Errorf("cuenta destino no encontrada: %s", in.ToAccount)
		}

		summary := fmt.Sprintf(
			"Va a transferir L %.2f a la cuenta %s (%s). ¿Confirma esta operación?",
			in.Amount, dest.AccountNumber, dest.FullName,
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: summary + " La operación no se ha ejecutado todavía. Use el botón de confirmación para completarla."}},
		}, TransferOutput{
			RequiresConfirmation: true,
			Amount:               in.Amount,
			Recipient: TransferRecipientInfo{
				AccountNumber: dest.AccountNumber,
				FullName:      dest.FullName,
			},
			Summary: summary,
		}, nil
	})
}

// ──────────────────────────────────────────────
// 6. get_transaction_history
// ──────────────────────────────────────────────

type GetTransactionHistoryInput struct {
	Limit uint32 `json:"limit,omitempty" jsonschema:"Cantidad máxima de transacciones a devolver (opcional, por defecto 20)"`
}

type GetTransactionHistoryOutput struct {
	Transactions []TransactionItem `json:"transactions"`
}

type TransactionItem struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
}

// ExecuteTransfer directly performs a transfer (bypassing the MCP
// "transfer" tool which only prepares confirmation). This is called by
// the chat confirmation handler after the user explicitly confirms.
func (f *ServerFactory) ExecuteTransfer(ctx context.Context, userID, toAccount string, amount float64) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("el monto debe ser positivo")
	}

	u, err := f.store.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("usuario no encontrado: %w", err)
	}

	if toAccount == u.AccountNumber {
		return "", fmt.Errorf("no puede transferirse dinero a sí mismo")
	}

	dest, err := f.store.GetUserByAccountNumber(ctx, toAccount)
	if err != nil {
		return "", fmt.Errorf("cuenta destino no encontrada: %s", toAccount)
	}

	if err := f.ledger.Transfer(u.TigerBeetleAccountID, dest.TigerBeetleAccountID, amount); err != nil {
		return "", fmt.Errorf("error al transferir: %w", err)
	}

	return fmt.Sprintf(
		"Transferencia exitosa. Se transfirieron L %.2f a la cuenta %s (%s).",
		amount, dest.AccountNumber, dest.FullName,
	), nil
}

func (f *ServerFactory) registerGetTransactionHistory(server *mcp.Server, userID string) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_transaction_history",
		Description: "Obtiene el historial de transacciones de la cuenta.",
		Title:       "Historial de transacciones",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in GetTransactionHistoryInput) (*mcp.CallToolResult, GetTransactionHistoryOutput, error) {
		u, err := f.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, GetTransactionHistoryOutput{}, fmt.Errorf("usuario no encontrado: %w", err)
		}

		limit := uint32(20)
		if in.Limit > 0 {
			limit = in.Limit
		}

		txs, err := f.ledger.GetHistory(u.TigerBeetleAccountID, limit)
		if err != nil {
			return nil, GetTransactionHistoryOutput{}, fmt.Errorf("error al obtener historial: %w", err)
		}

		items := make([]TransactionItem, len(txs))
		for i, tx := range txs {
			items[i] = TransactionItem{
				ID:          tx.ID,
				Description: tx.Description,
				Amount:      tx.Amount,
				Type:        tx.Type,
				Status:      tx.Status,
			}
		}

		text := "Historial de transacciones:\n"
		for _, item := range items {
			text += fmt.Sprintf("- %s: L %.2f (%s)\n", item.Type, item.Amount, item.Status)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, GetTransactionHistoryOutput{Transactions: items}, nil
	})
}
