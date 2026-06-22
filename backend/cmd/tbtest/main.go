package main

import (
	"fmt"
	"log"
	"os"

	"banca-backend/internal/config"
	"banca-backend/internal/ledger"
)

func main() {
	cfg := config.Load()

	addresses := []string{cfg.TigerBeetleAddress}

	l, err := ledger.NewLedger(cfg.TigerBeetleCluster, addresses)
	if err != nil {
		log.Fatalf("failed to create ledger: %v", err)
	}
	defer l.Close()

	accA, err := l.CreateUserAccount(0)
	if err != nil {
		log.Fatalf("failed to create account A: %v", err)
	}
	fmt.Printf("Account A: %s\n", accA)

	accB, err := l.CreateUserAccount(0)
	if err != nil {
		log.Fatalf("failed to create account B: %v", err)
	}
	fmt.Printf("Account B: %s\n", accB)

	if err := l.Deposit(accA, 100.00); err != nil {
		log.Fatalf("failed to deposit: %v", err)
	}
	fmt.Println("Deposited $100.00 into Account A")

	if err := l.Transfer(accA, accB, 30.00); err != nil {
		log.Fatalf("failed to transfer: %v", err)
	}
	fmt.Println("Transferred $30.00 from A to B")

	balA, err := l.GetBalance(accA)
	if err != nil {
		log.Fatalf("failed to get balance A: %v", err)
	}

	balB, err := l.GetBalance(accB)
	if err != nil {
		log.Fatalf("failed to get balance B: %v", err)
	}

	fmt.Printf("\nFinal balances:\n")
	fmt.Printf("  Account A: $%.2f\n", balA)
	fmt.Printf("  Account B: $%.2f\n", balB)

	if balA == 70.00 && balB == 30.00 {
		fmt.Println("\nSUCCESS: Balances match expected values ($70 and $30)")
		os.Exit(0)
	} else {
		fmt.Printf("\nUNEXPECTED: Expected $70.00 and $30.00, got $%.2f and $%.2f\n", balA, balB)
		os.Exit(1)
	}
}
