package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"banca-backend/internal/config"
	"banca-backend/internal/db"
	"banca-backend/internal/ledger"
	"banca-backend/internal/mcp"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := db.NewPostgresStore(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer store.Close()

	tb, err := ledger.NewLedger(cfg.TBClusterID, []string{cfg.TBAddress})
	if err != nil {
		log.Fatalf("tigerbeetle: %v", err)
	}

	factory := mcp.NewServerFactory(store, tb)

	// Use the first user in the database as our test user.
	// For a more robust test, create a dedicated test user.
	userID := findAnyUser(ctx, store)
	if userID == "" {
		log.Fatal("no users found in database; run seed first")
	}

	fmt.Printf("=== MCP Client Test ===\n")
	fmt.Printf("Testing with userID: %s\n\n", userID)

	// ── Test 1: get_balance ──
	fmt.Println("--- Test 1: get_balance ---")
	runToolTest(ctx, factory, userID, "get_balance", map[string]any{})

	// ── Test 2: get_account_info ──
	fmt.Println("\n--- Test 2: get_account_info ---")
	runToolTest(ctx, factory, userID, "get_account_info", map[string]any{})

	// ── Test 3: deposit $50 ──
	fmt.Println("\n--- Test 3: deposit (L 50.00) ---")
	runToolTest(ctx, factory, userID, "deposit", map[string]any{"amount": 50.0})

	// ── Test 4: withdraw $20 ──
	fmt.Println("\n--- Test 4: withdraw (L 20.00) ---")
	runToolTest(ctx, factory, userID, "withdraw", map[string]any{"amount": 20.0})

	// ── Test 5: transfer (should NOT execute, just return confirmation) ──
	fmt.Println("\n--- Test 5: transfer (L 10.00 to a dest) ---")
	runToolTest(ctx, factory, userID, "transfer", map[string]any{
		"to_account_number": "999-99999",
		"amount":            10.0,
	})

	// ── Test 6: get_transaction_history ──
	fmt.Println("\n--- Test 6: get_transaction_history ---")
	runToolTest(ctx, factory, userID, "get_transaction_history", map[string]any{"limit": 5})

	fmt.Println("\n=== All tests completed ===")
}

func runToolTest(ctx context.Context, factory *mcp.ServerFactory, userID, toolName string, args map[string]any) {
	server := factory.NewServer(userID)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	go func() {
		if err := server.Run(ctx, serverTransport); err != nil {
			log.Printf("MCP server exited: %v", err)
		}
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-test-client", Version: "v1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	for _, content := range result.Content {
		switch c := content.(type) {
		case *mcp.TextContent:
			fmt.Printf("Text: %s\n", c.Text)
		default:
			fmt.Printf("Content: %+v\n", c)
		}
	}

	if result.IsError {
		fmt.Printf("Tool returned an error\n")
	}
}

func findAnyUser(ctx context.Context, store *db.PostgresStore) string {
	// We don't have a "list all users" method, so try a known seed email
	emails := []string{
		"alice@example.com",
		"bob@example.com",
		"carol@example.com",
	}
	for _, email := range emails {
		u, err := store.GetUserByEmail(ctx, email)
		if err == nil {
			return u.ID
		}
	}
	return ""
}
