package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"banca-backend/internal/auth"
	"banca-backend/internal/db"
	"banca-backend/internal/ledger"
)

type SeedAccount struct {
	Email         string `json:"email"`
	FullName      string `json:"full_name"`
	Password      string `json:"password"`
	AccountNumber string `json:"account_number"`
	BalanceCents  uint64 `json:"balance_cents"`
}

func Load(ctx context.Context, store *db.PostgresStore, l *ledger.Ledger, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("seed file not found at %s, skipping", path)
			return nil
		}
		return fmt.Errorf("read seed file: %w", err)
	}

	var accounts []SeedAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return fmt.Errorf("parse seed file: %w", err)
	}

	for i, acc := range accounts {
		if err := seedOne(ctx, store, l, acc); err != nil {
			log.Printf("seed[%d] %s: %v", i, acc.Email, err)
		}
	}

	log.Printf("seed: processed %d accounts from %s", len(accounts), path)
	return nil
}

func seedOne(ctx context.Context, store *db.PostgresStore, l *ledger.Ledger, acc SeedAccount) error {
	exists, err := store.EmailExists(ctx, acc.Email)
	if err != nil {
		return fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil
	}

	passwordHash, err := auth.HashPassword(acc.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	tbAccountID, err := l.CreateUserAccount(acc.BalanceCents)
	if err != nil {
		return fmt.Errorf("create tb account: %w", err)
	}

	u := &db.User{
		Email:                acc.Email,
		PasswordHash:         passwordHash,
		FullName:             acc.FullName,
		TigerBeetleAccountID: tbAccountID,
		AccountNumber:        acc.AccountNumber,
	}

	if err := store.CreateUser(ctx, u); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	log.Printf("seed: created %s (acct %s, tb=%s)", acc.Email, acc.AccountNumber, tbAccountID)
	return nil
}
