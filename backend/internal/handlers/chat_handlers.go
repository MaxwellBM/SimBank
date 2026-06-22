package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"banca-backend/internal/ai"
	"banca-backend/internal/middleware"
	"banca-backend/internal/models"
)

type ChatHandler struct {
	ai *ai.ChatHandler
}

func NewChatHandler(aiHandler *ai.ChatHandler) *ChatHandler {
	return &ChatHandler{ai: aiHandler}
}

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if !h.ai.IsConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Chat IA no configurado. Agregue OPENROUTER_API_KEY a su archivo .env",
		})
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	orHistory := make([]ai.ORMessage, len(req.History))
	for i, m := range req.History {
		orHistory[i] = ai.ORMessage{Role: m.Role, Content: &m.Content}
	}

	replyText, pendingAction, requiresConf, err := h.ai.ProcessMessage(r.Context(), userID, req.Message, orHistory)
	if err != nil {
		log.Printf("chat error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	resp := models.ChatResponse{
		Reply:                replyText,
		RequiresConfirmation: requiresConf,
	}

	if pendingAction != nil {
		resp.Action = &models.PendingAction{
			ID:        pendingAction.ID,
			Type:      pendingAction.Type,
			FromAccount: pendingAction.FromAccount,
			ToAccount: pendingAction.ToAccount,
			Amount:    pendingAction.Amount,
			Status:    "pending",
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *ChatHandler) ConfirmAction(w http.ResponseWriter, r *http.Request) {
	if !h.ai.IsConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Chat IA no configurado. Agregue OPENROUTER_API_KEY a su archivo .env",
		})
		return
	}

	userID := r.Context().Value(middleware.UserIDKey).(string)

	var req models.ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ActionType != "transfer" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tipo de acción no soportada"})
		return
	}

	if req.ToAccount == "" || req.Amount <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "datos de transferencia incompletos"})
		return
	}

	result, err := h.ai.ExecuteConfirmedTransfer(r.Context(), userID, req.ToAccount, req.Amount)
	if err != nil {
		log.Printf("confirm transfer error: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "error al ejecutar la transferencia: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.ChatResponse{
		Reply: result,
	})
}
