package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	for i := range 10 {
		err = pool.Ping(ctx)
		if err == nil {
			log.Println("connected to postgres")
			return &PostgresStore{pool: pool}, nil
		}
		log.Printf("waiting for postgres (attempt %d/10): %v", i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	pool.Close()
	return nil, fmt.Errorf("postgres unreachable after 10 attempts: %w", err)
}

func (s *PostgresStore) Close() {
	s.pool.Close()
}

func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

// ──────────────────────────────────────────────
// Users
// ──────────────────────────────────────────────

func (s *PostgresStore) CreateUser(ctx context.Context, u *User) error {
	query := `
		INSERT INTO users (email, password_hash, full_name, tigerbeetle_account_id, account_number)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	err := s.pool.QueryRow(ctx, query,
		u.Email, u.PasswordHash, u.FullName, u.TigerBeetleAccountID, u.AccountNumber,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name,
		       tigerbeetle_account_id, account_number,
		       created_at, updated_at
		FROM users WHERE email = $1`

	u := &User{}
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.TigerBeetleAccountID, &u.AccountNumber,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name,
		       tigerbeetle_account_id, account_number,
		       created_at, updated_at
		FROM users WHERE id = $1`

	u := &User{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.TigerBeetleAccountID, &u.AccountNumber,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) GetUserByAccountNumber(ctx context.Context, accountNumber string) (*User, error) {
	query := `
		SELECT id, email, password_hash, full_name,
		       tigerbeetle_account_id, account_number,
		       created_at, updated_at
		FROM users WHERE account_number = $1`

	u := &User{}
	err := s.pool.QueryRow(ctx, query, accountNumber).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.TigerBeetleAccountID, &u.AccountNumber,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get user by account number: %w", err)
	}
	return u, nil
}

func (s *PostgresStore) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := s.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("email exists: %w", err)
	}
	return exists, nil
}

// ──────────────────────────────────────────────
// Sessions
// ──────────────────────────────────────────────

func (s *PostgresStore) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	_, err := s.pool.Exec(ctx, query, userID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *PostgresStore) IsSessionValid(ctx context.Context, tokenHash string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM sessions
			WHERE token_hash = $1
			  AND revoked = false
			  AND expires_at > NOW()
		)`
	var valid bool
	err := s.pool.QueryRow(ctx, query, tokenHash).Scan(&valid)
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	return valid, nil
}

func (s *PostgresStore) RevokeSession(ctx context.Context, tokenHash string) error {
	query := `UPDATE sessions SET revoked = true WHERE token_hash = $1`
	_, err := s.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

// User mirrors models.User but lives here to avoid circular imports.
type User struct {
	ID                   string
	Email                string
	PasswordHash         string
	FullName             string
	TigerBeetleAccountID string
	AccountNumber        string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
