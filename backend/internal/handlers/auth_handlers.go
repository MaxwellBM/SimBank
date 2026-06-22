package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"banca-backend/internal/auth"
	"banca-backend/internal/db"
	"banca-backend/internal/ledger"
	"banca-backend/internal/middleware"
	"banca-backend/internal/models"
)

type AuthHandler struct {
	store     *db.PostgresStore
	ledger    *ledger.Ledger
	jwtSecret string
}

func NewAuthHandler(store *db.PostgresStore, l *ledger.Ledger, jwtSecret string) *AuthHandler {
	return &AuthHandler{store: store, ledger: l, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email, password, and full_name are required"})
		return
	}

	exists, err := h.store.EmailExists(r.Context(), req.Email)
	if err != nil {
		log.Printf("check email: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("hash password: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	tbAccountID, err := h.ledger.CreateUserAccount(0)
	if err != nil {
		log.Printf("create tb account: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create account"})
		return
	}

	accountNumber := generateAccountNumber()

	u := &db.User{
		Email:                req.Email,
		PasswordHash:         passwordHash,
		FullName:             req.FullName,
		TigerBeetleAccountID: tbAccountID,
		AccountNumber:        accountNumber,
	}

	if err := h.store.CreateUser(r.Context(), u); err != nil {
		log.Printf("create user in db: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}

	token, err := auth.GenerateToken(u.ID, u.Email, h.jwtSecret)
	if err != nil {
		log.Printf("generate token: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	tokenHash := auth.HashToken(token)
	if err := h.store.CreateSession(r.Context(), u.ID, tokenHash, auth.ExpiryTime()); err != nil {
		log.Printf("create session: %v", err)
	}

	resp := models.AuthResponse{
		Token: token,
		User:  convertUser(u),
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	u, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}
		log.Printf("get user: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if !auth.CheckPassword(req.Password, u.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	token, err := auth.GenerateToken(u.ID, u.Email, h.jwtSecret)
	if err != nil {
		log.Printf("generate token: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	tokenHash := auth.HashToken(token)
	if err := h.store.CreateSession(r.Context(), u.ID, tokenHash, auth.ExpiryTime()); err != nil {
		log.Printf("create session: %v", err)
	}

	resp := models.AuthResponse{
		Token: token,
		User:  convertUser(u),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	if len(header) < 8 {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
		return
	}
	tokenString := header[7:]

	tokenHash := auth.HashToken(tokenString)
	if err := h.store.RevokeSession(r.Context(), tokenHash); err != nil {
		log.Printf("revoke session: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	u, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Printf("get user by id: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, convertUser(u))
}
