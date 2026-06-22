package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"banca-backend/internal/ai"
	"banca-backend/internal/config"
	"banca-backend/internal/db"
	"banca-backend/internal/handlers"
	"banca-backend/internal/ledger"
	"banca-backend/internal/mcp"
	"banca-backend/internal/middleware"
	"banca-backend/internal/seed"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := db.NewPostgresStore(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer store.Close()

	tb, err := ledger.NewLedger(cfg.TigerBeetleCluster, []string{cfg.TigerBeetleAddress})
	if err != nil {
		log.Fatalf("failed to connect to tigerbeetle: %v", err)
	}

	if err := seed.Load(ctx, store, tb, cfg.SeedDataPath); err != nil {
		log.Printf("seed warning: %v", err)
	}

	authH := handlers.NewAuthHandler(store, tb, cfg.JWTSecret)
	bankH := handlers.NewBankHandler(store, tb)
	authMw := middleware.AuthMiddleware(store, cfg.JWTSecret)

	mcpFactory := mcp.NewServerFactory(store, tb)
	aiChat := ai.NewChatHandler(mcpFactory, cfg.OpenRouterAPIKey, cfg.AIModel)
	chatH := handlers.NewChatHandler(aiChat)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(corsMiddleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"service":"SimBank API","version":"0.1.0"}`)
	})

	r.Post("/api/auth/register", authH.Register)
	r.Post("/api/auth/login", authH.Login)
	r.Post("/api/auth/logout", authH.Logout)

	r.Route("/api", func(r chi.Router) {
		r.Use(authMw)

		r.Get("/account", bankH.AccountInfo)
		r.Get("/account/balance", bankH.Balance)
		r.Get("/account/me", authH.Me)

		r.Post("/transactions/deposit", bankH.Deposit)
		r.Post("/transactions/withdraw", bankH.Withdraw)
		r.Post("/transactions/transfer", bankH.Transfer)
		r.Get("/transactions/history", bankH.History)

		r.Post("/chat", chatH.Chat)
		r.Post("/chat/confirm", chatH.ConfirmAction)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("SimBank backend starting on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-shutdown
	log.Println("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
