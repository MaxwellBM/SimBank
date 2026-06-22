package ledger

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"strconv"
	"time"

	tb "github.com/tigerbeetle/tigerbeetle-go"

	"banca-backend/internal/models"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

const (
	bankControlAccountID = 1
	ledgerID             = 1
	transferCode         = 1
	centsPerDollar       = 100
)

// resolveAddresses converts host:port addresses to IP:port, because the TB
// native client (v0.17.x) rejects hostnames and requires raw IP addresses.
func resolveAddresses(addrs []string) ([]string, error) {
	resolved := make([]string, len(addrs))
	for i, addr := range addrs {
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("split %q: %w", addr, err)
		}
		if _, err := strconv.Atoi(portStr); err != nil {
			return nil, fmt.Errorf("invalid port in %q: %w", addr, err)
		}
		// If already an IP, use it directly.
		if net.ParseIP(host) != nil {
			resolved[i] = addr
			continue
		}
		ips, err := net.LookupHost(host)
		if err != nil {
			return nil, fmt.Errorf("lookup %q: %w", host, err)
		}
		resolved[i] = net.JoinHostPort(ips[0], portStr)
	}
	return resolved, nil
}

type Ledger struct {
	client tb.Client
}

func NewLedger(clusterID uint32, addresses []string) (*Ledger, error) {
	resolved, err := resolveAddresses(addresses)
	if err != nil {
		return nil, fmt.Errorf("resolve addresses: %w", err)
	}

	var client tb.Client

	for i := 0; i < 30; i++ {
		client, err = tb.NewClient(tb.ToUint128(uint64(clusterID)), resolved)
		if err == nil {
			break
		}
		if i < 29 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("connect to tigerbeetle after retries: %w", err)
	}

	l := &Ledger{client: client}

	if err := l.ensureBankControlAccount(); err != nil {
		client.Close()
		return nil, fmt.Errorf("bank control account: %w", err)
	}

	return l, nil
}

func (l *Ledger) Close() {
	l.client.Close()
}

func (l *Ledger) ensureBankControlAccount() error {
	bankID := tb.ToUint128(bankControlAccountID)
	accounts, err := l.client.LookupAccounts([]tb.Uint128{bankID})
	if err != nil {
		return fmt.Errorf("lookup bank account: %w", err)
	}
	if len(accounts) > 0 {
		return nil
	}

	results, err := l.client.CreateAccounts([]tb.Account{{
		ID:     bankID,
		Ledger: ledgerID,
		Code:   transferCode,
	}})
	if err != nil {
		return err
	}

	if len(results) > 0 && results[0].Status != tb.AccountCreated {
		if results[0].Status != tb.AccountExists {
			return fmt.Errorf("create bank account: %s", results[0].Status)
		}
	}

	return nil
}

func (l *Ledger) CreateUserAccount(initialBalanceCents uint64) (string, error) {
	accountID := tb.ID()

	flags := tb.AccountFlags{
		DebitsMustNotExceedCredits: true,
		History:                    true,
	}

	account := tb.Account{
		ID:     accountID,
		Ledger: ledgerID,
		Code:   transferCode,
		Flags:  flags.ToUint16(),
	}

	results, err := l.client.CreateAccounts([]tb.Account{account})
	if err != nil {
		return "", fmt.Errorf("create account: %w", err)
	}

	if len(results) > 0 && results[0].Status != tb.AccountCreated {
		return "", fmt.Errorf("create account: %s", results[0].Status)
	}

	if initialBalanceCents > 0 {
		if err := l.depositCents(accountID, initialBalanceCents); err != nil {
			return "", fmt.Errorf("initial deposit: %w", err)
		}
	}

	return accountID.String(), nil
}

func (l *Ledger) GetBalance(accountID string) (float64, error) {
	id, err := parseAccountID(accountID)
	if err != nil {
		return 0, err
	}

	accounts, err := l.client.LookupAccounts([]tb.Uint128{id})
	if err != nil {
		return 0, fmt.Errorf("lookup account: %w", err)
	}

	if len(accounts) == 0 {
		return 0, errors.New("account not found")
	}

	balance := balanceFromAccount(accounts[0])
	return bigIntToDollars(balance), nil
}

func (l *Ledger) Deposit(accountID string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	id, err := parseAccountID(accountID)
	if err != nil {
		return err
	}

	cents := dollarsToCents(amount)
	return l.depositCents(id, cents)
}

func (l *Ledger) depositCents(accountID tb.Uint128, cents uint64) error {
	transfer := tb.Transfer{
		ID:              tb.ID(),
		DebitAccountID:  tb.ToUint128(bankControlAccountID),
		CreditAccountID: accountID,
		Amount:          tb.ToUint128(cents),
		Ledger:          ledgerID,
		Code:            transferCode,
	}

	results, err := l.client.CreateTransfers([]tb.Transfer{transfer})
	if err != nil {
		return fmt.Errorf("create transfer: %w", err)
	}

	if len(results) > 0 && results[0].Status != tb.TransferCreated {
		return fmt.Errorf("deposit failed: %s", results[0].Status)
	}

	return nil
}

func (l *Ledger) Withdraw(accountID string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	id, err := parseAccountID(accountID)
	if err != nil {
		return err
	}

	cents := dollarsToCents(amount)

	transfer := tb.Transfer{
		ID:              tb.ID(),
		DebitAccountID:  id,
		CreditAccountID: tb.ToUint128(bankControlAccountID),
		Amount:          tb.ToUint128(cents),
		Ledger:          ledgerID,
		Code:            transferCode,
	}

	results, err := l.client.CreateTransfers([]tb.Transfer{transfer})
	if err != nil {
		return fmt.Errorf("create transfer: %w", err)
	}

	if len(results) > 0 && results[0].Status != tb.TransferCreated {
		if results[0].Status == tb.TransferExceedsCredits {
			return fmt.Errorf("%w: withdrawal of $%.2f would overdraw account", ErrInsufficientFunds, amount)
		}
		return fmt.Errorf("withdraw failed: %s", results[0].Status)
	}

	return nil
}

func (l *Ledger) Transfer(fromAccountID, toAccountID string, amount float64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}

	from, err := parseAccountID(fromAccountID)
	if err != nil {
		return err
	}

	to, err := parseAccountID(toAccountID)
	if err != nil {
		return err
	}

	cents := dollarsToCents(amount)

	transfer := tb.Transfer{
		ID:              tb.ID(),
		DebitAccountID:  from,
		CreditAccountID: to,
		Amount:          tb.ToUint128(cents),
		Ledger:          ledgerID,
		Code:            transferCode,
	}

	results, err := l.client.CreateTransfers([]tb.Transfer{transfer})
	if err != nil {
		return fmt.Errorf("create transfer: %w", err)
	}

	if len(results) > 0 && results[0].Status != tb.TransferCreated {
		if results[0].Status == tb.TransferExceedsCredits {
			return fmt.Errorf("%w: transfer of $%.2f would overdraw account", ErrInsufficientFunds, amount)
		}
		return fmt.Errorf("transfer failed: %s", results[0].Status)
	}

	return nil
}

func (l *Ledger) GetHistory(accountID string, limit uint32) ([]models.Transaction, error) {
	id, err := parseAccountID(accountID)
	if err != nil {
		return nil, err
	}

	filter := tb.AccountFilter{
		AccountID: id,
		Limit:     limit,
		Flags: tb.AccountFilterFlags{
			Debits:   true,
			Credits:  true,
			Reversed: true,
		}.ToUint32(),
	}

	transfers, err := l.client.GetAccountTransfers(filter)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	txs := make([]models.Transaction, len(transfers))
	for i, t := range transfers {
		txs[i] = transferToTransaction(t)
	}

	return txs, nil
}

func parseAccountID(s string) (tb.Uint128, error) {
	return tb.HexStringToUint128(s)
}

func dollarsToCents(amount float64) uint64 {
	return uint64(math.Round(amount * centsPerDollar))
}

func bigIntToDollars(n *big.Int) float64 {
	return float64(n.Uint64()) / centsPerDollar
}

func balanceFromAccount(a tb.Account) *big.Int {
	credits := a.CreditsPosted.BigInt()
	debits := a.DebitsPosted.BigInt()
	return new(big.Int).Sub(credits, debits)
}

func transferToTransaction(t tb.Transfer) models.Transaction {
	bankID := tb.ToUint128(bankControlAccountID)
	amountCents := t.Amount.BigInt().Uint64()

	var txType string
	if t.DebitAccountID == bankID {
		txType = "deposit"
	} else if t.CreditAccountID == bankID {
		txType = "withdrawal"
	} else {
		txType = "transfer"
	}

	return models.Transaction{
		ID:          t.ID.String(),
		FromAccount: t.DebitAccountID.String(),
		ToAccount:   t.CreditAccountID.String(),
		Amount:      float64(amountCents) / centsPerDollar,
		Type:        txType,
		Description: "",
		Status:      "completed",
		Timestamp:   time.Unix(0, int64(t.Timestamp)),
	}
}
