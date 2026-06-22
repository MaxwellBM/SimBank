package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"banca-backend/internal/auth"
	"banca-backend/internal/config"
	"banca-backend/internal/db"
	"banca-backend/internal/ledger"
)

type BulkUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FullName  string `json:"full_name"`
	CreatedAt string `json:"created_at"`
}

type result struct {
	Email string
	Err   error
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Uso: go run ./cmd/bulk-import <archivo.json>")
	}
	path := os.Args[1]

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	store, err := db.NewPostgresStore(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer store.Close()

	tb, err := ledger.NewLedger(cfg.TigerBeetleCluster, []string{cfg.TigerBeetleAddress})
	if err != nil {
		log.Fatalf("tigerbeetle: %v", err)
	}
	defer tb.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("leer archivo: %v", err)
	}

	// Try flat array first, then {"users": [...]}
	var users []BulkUser
	if err := json.Unmarshal(data, &users); err != nil {
		var wrapped struct {
			Users []BulkUser `json:"users"`
		}
		if err2 := json.Unmarshal(data, &wrapped); err2 != nil {
			log.Fatalf("parsear JSON (intenté array y {users: []}): primer error: %v", err)
		}
		users = wrapped.Users
	}

	log.Printf("Cargando %d usuarios desde %s", len(users), path)

	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}

	jobs := make(chan BulkUser, numWorkers*2)
	results := make(chan result, numWorkers*2)
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Add(1)
		go worker(ctx, store, tb, jobs, results, &wg)
	}

	go func() {
		for _, u := range users {
			jobs <- u
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var ok, fails int
	for r := range results {
		if r.Err != nil {
			log.Printf("✗ %s: %v", r.Email, r.Err)
			fails++
		} else {
			ok++
		}
	}

	log.Printf("Importación completada: %d exitosos, %d fallos de %d total", ok, fails, len(users))
}

func worker(ctx context.Context, store *db.PostgresStore, tb *ledger.Ledger, jobs <-chan BulkUser, results chan<- result, wg *sync.WaitGroup) {
	defer wg.Done()
	for u := range jobs {
		err := importOne(ctx, store, tb, u)
		results <- result{Email: u.Email, Err: err}
	}
}

func importOne(ctx context.Context, store *db.PostgresStore, tb *ledger.Ledger, u BulkUser) error {
	exists, err := store.EmailExists(ctx, u.Email)
	if err != nil {
		return fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil
	}

	passwordHash, err := auth.HashPassword(u.Password)
	if err != nil {
		return fmt.Errorf("hash: %w", err)
	}

	tbAccountID, err := tb.CreateUserAccount(0)
	if err != nil {
		return fmt.Errorf("tb account: %w", err)
	}

	accountNumber := generateAccountNumber()

	var createdAt time.Time
	if u.CreatedAt != "" {
		createdAt, err = time.Parse(time.RFC3339, u.CreatedAt)
		if err != nil {
			createdAt = time.Now()
		}
	} else {
		createdAt = time.Now()
	}

	record := &db.User{
		ID:                   u.ID,
		Email:                u.Email,
		PasswordHash:         passwordHash,
		FullName:             u.FullName,
		TigerBeetleAccountID: tbAccountID,
		AccountNumber:        accountNumber,
		CreatedAt:            createdAt,
	}

	if err := store.CreateUserPreserveID(ctx, record); err != nil {
		return fmt.Errorf("db: %w", err)
	}

	log.Printf("✓ %s → cuenta %s, tb=%s", u.Email, accountNumber, tbAccountID[:12]+"…")
	return nil
}

func generateAccountNumber() string {
	return fmt.Sprintf("%03d-%05d", time.Now().UnixMilli()%900+100, time.Now().UnixNano()%90000+10000)
}
