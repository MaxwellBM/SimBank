package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strconv"

	"banca-backend/internal/db"
	"banca-backend/internal/ledger"
	"banca-backend/internal/middleware"
	"banca-backend/internal/models"
)

type BankHandler struct {
	store  *db.PostgresStore
	ledger *ledger.Ledger
}

func NewBankHandler(store *db.PostgresStore, l *ledger.Ledger) *BankHandler {
	return &BankHandler{store: store, ledger: l}
}

func (h *BankHandler) AccountInfo(w http.ResponseWriter, r *http.Request) {
	u, err := h.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	balance, err := h.ledger.GetBalance(u.TigerBeetleAccountID)
	if err != nil {
		log.Printf("get balance: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get balance"})
		return
	}

	writeJSON(w, http.StatusOK, models.AccountInfo{
		AccountNumber: u.AccountNumber,
		Balance:       balance,
		Currency:      "HNL",
		AccountType:   "checking",
	})
}

func (h *BankHandler) Balance(w http.ResponseWriter, r *http.Request) {
	u, err := h.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	balance, err := h.ledger.GetBalance(u.TigerBeetleAccountID)
	if err != nil {
		log.Printf("get balance: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get balance"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]float64{"balance": balance})
}

func (h *BankHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	u, err := h.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	var req models.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}

	if err := h.ledger.Deposit(u.TigerBeetleAccountID, req.Amount); err != nil {
		log.Printf("deposit: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "deposit failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deposit successful"})
}

func (h *BankHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	u, err := h.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	var req models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}

	if err := h.ledger.Withdraw(u.TigerBeetleAccountID, req.Amount); err != nil {
		if errors.Is(err, ledger.ErrInsufficientFunds) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("withdraw: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "withdrawal failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "withdrawal successful"})
}

func (h *BankHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	u, err := h.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	var req models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "amount must be positive"})
		return
	}

	if req.ToAccount == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "to_account is required"})
		return
	}

	if req.ToAccount == u.AccountNumber {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot transfer to your own account"})
		return
	}

	dest, err := h.store.GetUserByAccountNumber(r.Context(), req.ToAccount)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "destination account not found"})
		return
	}

	if err := h.ledger.Transfer(u.TigerBeetleAccountID, dest.TigerBeetleAccountID, req.Amount); err != nil {
		if errors.Is(err, ledger.ErrInsufficientFunds) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		log.Printf("transfer: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "transfer failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "transfer successful"})
}

func (h *BankHandler) History(w http.ResponseWriter, r *http.Request) {
	u, err := h.authUser(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	limit := uint32(20)
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.ParseUint(l, 10, 32); err == nil && n > 0 {
			limit = uint32(n)
		}
	}

	txs, err := h.ledger.GetHistory(u.TigerBeetleAccountID, limit)
	if err != nil {
		log.Printf("get history: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get history"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"transactions": txs})
}

func (h *BankHandler) authUser(r *http.Request) (*db.User, error) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	return h.store.GetUserByID(r.Context(), userID)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func convertUser(u *db.User) models.User {
	return models.User{
		ID:                   u.ID,
		Email:                u.Email,
		FullName:             u.FullName,
		TigerBeetleAccountID: u.TigerBeetleAccountID,
		AccountNumber:        u.AccountNumber,
		CreatedAt:            u.CreatedAt,
		UpdatedAt:            u.UpdatedAt,
	}
}

func generateAccountNumber() string {
	prefix := rand.Intn(900) + 100
	suffix := rand.Intn(90000) + 10000
	return strconv.Itoa(prefix) + "-" + strconv.Itoa(suffix)
}
