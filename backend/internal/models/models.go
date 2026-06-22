package models

import "time"

type User struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	PasswordHash         string    `json:"-"`
	FullName             string    `json:"full_name"`
	TigerBeetleAccountID string    `json:"tigerbeetle_account_id"`
	AccountNumber        string    `json:"account_number"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type AccountInfo struct {
	AccountNumber string  `json:"account_number"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
	AccountType   string  `json:"account_type"`
}

type DepositRequest struct {
	AccountNumber string  `json:"account_number"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
}

type WithdrawRequest struct {
	AccountNumber string  `json:"account_number"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
}

type TransferRequest struct {
	FromAccount string  `json:"from_account"`
	ToAccount   string  `json:"to_account"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type Transaction struct {
	ID          string    `json:"id"`
	FromAccount string    `json:"from_account"`
	ToAccount   string    `json:"to_account"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Reply  string         `json:"reply"`
	Action *PendingAction `json:"action,omitempty"`
}

type PendingAction struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	FromAccount string  `json:"from_account,omitempty"`
	ToAccount   string  `json:"to_account,omitempty"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description,omitempty"`
	Status      string  `json:"status"`
}
