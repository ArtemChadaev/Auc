package user

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBURL = "postgresql://test:password@localhost:5432/db"

func loadEnv() {
	// Traverse up to find .env
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		if f, err := os.Open(envPath); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
					if os.Getenv(key) == "" {
						_ = os.Setenv(key, val)
					}
				}
			}
			_ = f.Close()
			break
		}
		dir = filepath.Dir(dir)
	}
	_ = cfg.Init()
}

func setupUserTest(t *testing.T) (*Service, *pgxpool.Pool, *valkey.Client) {
	t.Helper()
	loadEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDBURL)
	if err != nil {
		t.Fatalf("failed to connect to test postgres: %v", err)
	}

	valkeyClient, err := valkey.NewClient(ctx, valkey.Config{Host: "localhost", Port: "6379"})
	if err != nil {
		t.Fatalf("failed to connect to test valkey: %v", err)
	}

	s3Client, err := storageS3.NewClient(ctx, storageS3.Config{
		Endpoint:  "https://s3.ru1.storage.beget.cloud",
		Region:    "ru1",
		AccessKey: "test",
		SecretKey: "test",
		Bucket:    "auc-test",
	})
	if err != nil {
		t.Fatalf("failed to initialize s3 client: %v", err)
	}

	repo := NewRepo(pool)
	service := NewService(repo, pool, s3Client, valkeyClient)

	t.Cleanup(func() {
		pool.Close()
		_ = valkeyClient.Close()
	})

	return service, pool, valkeyClient
}

func createTestUser(t *testing.T, pool *pgxpool.Pool, balance, hold int) (uuid.UUID, string, string) {
	t.Helper()
	ctx := context.Background()
	uid := uuid.New()
	email := fmt.Sprintf("test_%s@example.com", strings.ReplaceAll(uid.String(), "-", ""))
	pass := "Password123!"
	passHash, err := createPassword(pass)
	if err != nil {
		t.Fatalf("createPassword failed: %v", err)
	}

	name := fmt.Sprintf("User_%s", uid.String()[:8])
	_, err = pool.Exec(ctx, `
		INSERT INTO users (id, name, email, password_hash, balance, hold, role)
		VALUES ($1, $2, $3, $4, $5, $6, 'user')
	`, uid, name, email, passHash, balance, hold)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	return uid, email, pass
}

// 1. Password creation & verification
func TestPassword_CreateAndCheck(t *testing.T) {
	password := "SecretPass123!@#"

	hash, err := createPassword(password)
	if err != nil {
		t.Fatalf("createPassword returned error: %v", err)
	}
	if len(hash) == 0 {
		t.Fatal("createPassword returned empty hash")
	}

	// Verify correct password
	if err := checkPassword(hash, password); err != nil {
		t.Errorf("checkPassword failed for valid password: %v", err)
	}

	// Verify wrong password returns error
	if err := checkPassword(hash, "WrongPassword123!"); err == nil {
		t.Errorf("checkPassword succeeded for incorrect password, expected error")
	}
}

// 2. Claim Free Coin:
// - Increases balance by 1000
// - Once per calendar day (UTC)
// - Second call on same day returns ErrFreeCoinLimitReached
func TestClaimFreeCoin_Success(t *testing.T) {
	service, pool, valkeyClient := setupUserTest(t)
	ctx := context.Background()

	initialBalance := 200
	uid, _, _ := createTestUser(t, pool, initialBalance, 0)

	// Clean up any valkey keys for this user if needed
	_ = valkeyClient.Raw().Del(ctx, fmt.Sprintf("free_coin:%s", uid.String())).Err()

	userRes, err := service.claimFreeCoin(ctx, uid)
	if err != nil {
		t.Fatalf("claimFreeCoin failed: %v", err)
	}

	expectedBalance := initialBalance + 1000
	if userRes.Balance != expectedBalance {
		t.Errorf("expected returned user balance %d, got %d", expectedBalance, userRes.Balance)
	}

	// Check DB
	var dbBalance int
	err = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", uid).Scan(&dbBalance)
	if err != nil {
		t.Fatalf("failed to query user balance from DB: %v", err)
	}
	if dbBalance != expectedBalance {
		t.Errorf("expected DB balance %d, got %d", expectedBalance, dbBalance)
	}
}

func TestClaimFreeCoin_DailyLimit(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	uid, _, _ := createTestUser(t, pool, 0, 0)

	// First claim should succeed
	_, err := service.claimFreeCoin(ctx, uid)
	if err != nil {
		t.Fatalf("first claimFreeCoin failed: %v", err)
	}

	// Second claim on same day must fail with ErrFreeCoinLimitReached
	_, err = service.claimFreeCoin(ctx, uid)
	if err == nil {
		t.Fatal("second claimFreeCoin on same day succeeded, expected ErrFreeCoinLimitReached")
	}
	if !errors.Is(err, ErrFreeCoinLimitReached) {
		t.Errorf("expected error wrapping ErrFreeCoinLimitReached, got: %v", err)
	}
}

// 3. Change Balance:
// - Positive change increases balance
// - Negative change decreases balance
func TestChangeBalance_PositiveAndNegative(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	initialBalance := 1000
	uid, _, _ := createTestUser(t, pool, initialBalance, 0)

	// Positive change
	err := service.changeBalance(ctx, uid, 500)
	if err != nil {
		t.Fatalf("changeBalance(+500) failed: %v", err)
	}

	var balance int
	err = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", uid).Scan(&balance)
	if err != nil {
		t.Fatalf("failed to query balance: %v", err)
	}
	if balance != 1500 {
		t.Errorf("expected balance 1500, got %d", balance)
	}

	// Negative change
	err = service.changeBalance(ctx, uid, -300)
	if err != nil {
		t.Fatalf("changeBalance(-300) failed: %v", err)
	}

	err = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", uid).Scan(&balance)
	if err != nil {
		t.Fatalf("failed to query balance: %v", err)
	}
	if balance != 1200 {
		t.Errorf("expected balance 1200, got %d", balance)
	}
}

// 4. Registration and Password Authentication
func TestRegister_And_AuthPassword(t *testing.T) {
	service, _, _ := setupUserTest(t)
	ctx := context.Background()

	newUID := uuid.New()
	email := fmt.Sprintf("reg_%s@example.com", strings.ReplaceAll(newUID.String(), "-", ""))
	password := "StrongPass123!"
	name := "Register Test"
	dev := Device{OS: "macOS", Browser: "Chrome"}

	// 1. Register with explicit UID
	tokens, err := service.register(ctx, &newUID, name, email, password, dev)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Errorf("expected non-empty AccessToken")
	}
	if tokens.RefreshToken == "" {
		t.Errorf("expected non-empty RefreshToken")
	}

	// 2. Register with nil UID (auto-generated)
	email2 := fmt.Sprintf("reg_nil_%s@example.com", strings.ReplaceAll(uuid.New().String(), "-", ""))
	tokens2, err := service.register(ctx, nil, name, email2, password, dev)
	if err != nil {
		t.Fatalf("register with nil uid failed: %v", err)
	}
	if tokens2.AccessToken == "" || tokens2.RefreshToken == "" {
		t.Errorf("register with nil uid returned empty tokens")
	}

	// 3. Auth with valid credentials
	authTokens, err := service.authPassword(ctx, email, password, dev)
	if err != nil {
		t.Fatalf("authPassword failed with valid credentials: %v", err)
	}
	if authTokens.AccessToken == "" || authTokens.RefreshToken == "" {
		t.Errorf("authPassword returned empty tokens")
	}

	// 4. Auth with invalid password
	_, err = service.authPassword(ctx, email, "WrongPassword!", dev)
	if err == nil {
		t.Errorf("authPassword succeeded with wrong password, expected error")
	}

	// 5. Auth with non-existent email
	_, err = service.authPassword(ctx, "nonexistent@example.com", password, dev)
	if err == nil {
		t.Errorf("authPassword succeeded with non-existent email, expected error")
	}
}

// 5. Auth Refresh
func TestAuthRefresh(t *testing.T) {
	service, _, _ := setupUserTest(t)
	ctx := context.Background()

	uid := uuid.New()
	email := fmt.Sprintf("refresh_%s@example.com", strings.ReplaceAll(uid.String(), "-", ""))
	password := "StrongPass123!"
	dev := Device{OS: "macOS", Browser: "Chrome"}

	regTokens, err := service.register(ctx, &uid, "Refresh User", email, password, dev)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Refresh with valid refresh token
	newTokens, err := service.authRefresh(ctx, regTokens.RefreshToken)
	if err != nil {
		t.Fatalf("authRefresh failed with valid refresh token: %v", err)
	}
	if newTokens.AccessToken == "" || newTokens.RefreshToken == "" {
		t.Errorf("authRefresh returned empty tokens")
	}

	// Refresh with invalid token
	_, err = service.authRefresh(ctx, "invalid-refresh-token")
	if err == nil {
		t.Errorf("authRefresh succeeded with invalid token, expected error")
	}
}

// -----------------------------------------------------------------------------
// 6. AdminChangeBalance Tests
// -----------------------------------------------------------------------------

func TestAdminChangeBalance_TopUp(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	initialBalance := 200
	uid, _, _ := createTestUser(t, pool, initialBalance, 0)

	newBal, err := service.AdminChangeBalance(ctx, uid, 500)
	if err != nil {
		t.Fatalf("AdminChangeBalance(+500) failed: %v", err)
	}
	if newBal != 700 {
		t.Errorf("expected returned balance 700, got %d", newBal)
	}

	var dbBal int
	err = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", uid).Scan(&dbBal)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if dbBal != 700 {
		t.Errorf("expected DB balance 700, got %d", dbBal)
	}

	var ledgerAmount, balanceAfter int
	var ledgerType string
	err = pool.QueryRow(ctx, `
		SELECT amount, balance_after, type 
		FROM ledger_entries 
		WHERE user_id = $1 
		ORDER BY id DESC LIMIT 1
	`, uid).Scan(&ledgerAmount, &balanceAfter, &ledgerType)
	if err != nil {
		t.Fatalf("failed to query ledger_entries: %v", err)
	}
	if ledgerAmount != 500 {
		t.Errorf("expected ledger amount 500, got %d", ledgerAmount)
	}
	if balanceAfter != 700 {
		t.Errorf("expected ledger balance_after 700, got %d", balanceAfter)
	}
	if ledgerType != "top_up" {
		t.Errorf("expected ledger type 'top_up', got %q", ledgerType)
	}
}

func TestAdminChangeBalance_Deduct(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	initialBalance := 1000
	uid, _, _ := createTestUser(t, pool, initialBalance, 0)

	newBal, err := service.AdminChangeBalance(ctx, uid, -300)
	if err != nil {
		t.Fatalf("AdminChangeBalance(-300) failed: %v", err)
	}
	if newBal != 700 {
		t.Errorf("expected returned balance 700, got %d", newBal)
	}

	var dbBal int
	err = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", uid).Scan(&dbBal)
	if err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if dbBal != 700 {
		t.Errorf("expected DB balance 700, got %d", dbBal)
	}

	var ledgerAmount, balanceAfter int
	err = pool.QueryRow(ctx, `
		SELECT amount, balance_after 
		FROM ledger_entries 
		WHERE user_id = $1 
		ORDER BY id DESC LIMIT 1
	`, uid).Scan(&ledgerAmount, &balanceAfter)
	if err != nil {
		t.Fatalf("failed to query ledger_entries: %v", err)
	}
	if ledgerAmount != -300 {
		t.Errorf("expected ledger amount -300, got %d", ledgerAmount)
	}
	if balanceAfter != 700 {
		t.Errorf("expected ledger balance_after 700, got %d", balanceAfter)
	}
}

func TestAdminChangeBalance_DeductBelowHold(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	uid, _, _ := createTestUser(t, pool, 500, 400)

	_, err := service.AdminChangeBalance(ctx, uid, -150)
	if err == nil {
		t.Errorf("AdminChangeBalance(-150) succeeded, expected ErrInsufficientBalance")
	} else if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance, got: %v", err)
	}
}

func TestAdminChangeBalance_ZeroAmount(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	uid, _, _ := createTestUser(t, pool, 500, 0)

	_, err := service.AdminChangeBalance(ctx, uid, 0)
	if err == nil {
		t.Errorf("AdminChangeBalance(0) succeeded, expected ErrInvalidAmount")
	} else if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount, got: %v", err)
	}
}

func TestAdminChangeBalance_NonExistentUser(t *testing.T) {
	service, _, _ := setupUserTest(t)
	ctx := context.Background()

	nonExistentUID := uuid.New()
	_, err := service.AdminChangeBalance(ctx, nonExistentUID, 100)
	if err == nil {
		t.Errorf("AdminChangeBalance for non-existent user succeeded, expected error")
	}
}

// -----------------------------------------------------------------------------
// 7. AdminTransferFunds Tests
// -----------------------------------------------------------------------------

func TestAdminTransferFunds_Success(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	userA, _, _ := createTestUser(t, pool, 1000, 0)
	userB, _, _ := createTestUser(t, pool, 200, 0)

	err := service.AdminTransferFunds(ctx, userA, userB, 350)
	if err != nil {
		t.Fatalf("AdminTransferFunds failed: %v", err)
	}

	var balA, balB int
	_ = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", userA).Scan(&balA)
	_ = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", userB).Scan(&balB)

	if balA != 650 {
		t.Errorf("expected user A balance 650, got %d", balA)
	}
	if balB != 550 {
		t.Errorf("expected user B balance 550, got %d", balB)
	}

	var amountA, amountB int
	err = pool.QueryRow(ctx, "SELECT amount FROM ledger_entries WHERE user_id = $1 ORDER BY id DESC LIMIT 1", userA).Scan(&amountA)
	if err != nil {
		t.Fatalf("failed to query ledger entry for user A: %v", err)
	}
	if amountA != -350 {
		t.Errorf("expected ledger amount -350 for user A, got %d", amountA)
	}

	err = pool.QueryRow(ctx, "SELECT amount FROM ledger_entries WHERE user_id = $1 ORDER BY id DESC LIMIT 1", userB).Scan(&amountB)
	if err != nil {
		t.Fatalf("failed to query ledger entry for user B: %v", err)
	}
	if amountB != 350 {
		t.Errorf("expected ledger amount +350 for user B, got %d", amountB)
	}
}

func TestAdminTransferFunds_InsufficientBalance(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	userA, _, _ := createTestUser(t, pool, 500, 400)
	userB, _, _ := createTestUser(t, pool, 200, 0)

	err := service.AdminTransferFunds(ctx, userA, userB, 150)
	if err == nil {
		t.Errorf("AdminTransferFunds with insufficient balance succeeded, expected ErrInsufficientBalance")
	} else if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance, got: %v", err)
	}
}

func TestAdminTransferFunds_SameUser(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	userA, _, _ := createTestUser(t, pool, 500, 0)

	err := service.AdminTransferFunds(ctx, userA, userA, 100)
	if err == nil {
		t.Errorf("AdminTransferFunds to same user succeeded, expected ErrSameUser")
	} else if !errors.Is(err, ErrSameUser) {
		t.Errorf("expected ErrSameUser, got: %v", err)
	}
}

func TestAdminTransferFunds_InvalidAmount(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	userA, _, _ := createTestUser(t, pool, 500, 0)
	userB, _, _ := createTestUser(t, pool, 500, 0)

	// amount = 0
	err := service.AdminTransferFunds(ctx, userA, userB, 0)
	if err == nil {
		t.Errorf("AdminTransferFunds(0) succeeded, expected ErrInvalidAmount")
	} else if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount for amount=0, got: %v", err)
	}

	// amount < 0
	err = service.AdminTransferFunds(ctx, userA, userB, -50)
	if err == nil {
		t.Errorf("AdminTransferFunds(-50) succeeded, expected ErrInvalidAmount")
	} else if !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("expected ErrInvalidAmount for amount=-50, got: %v", err)
	}
}

func TestAdminTransferFunds_ConcurrentDeadlock(t *testing.T) {
	service, pool, _ := setupUserTest(t)
	ctx := context.Background()

	userA, _, _ := createTestUser(t, pool, 100000, 0)
	userB, _, _ := createTestUser(t, pool, 100000, 0)

	const n = 20
	errCh := make(chan error, n*2)

	for i := 0; i < n; i++ {
		go func() {
			errCh <- service.AdminTransferFunds(ctx, userA, userB, 10)
		}()
		go func() {
			errCh <- service.AdminTransferFunds(ctx, userB, userA, 10)
		}()
	}

	for i := 0; i < n*2; i++ {
		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("concurrent transfer error: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("deadlock detected during concurrent transfers")
		}
	}
}
