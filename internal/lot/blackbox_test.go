package lot

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/ArtemChadaev/Auction/cmd/cfg"
	"github.com/ArtemChadaev/Auction/cmd/storageS3"
	"github.com/ArtemChadaev/Auction/cmd/valkey"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBURL = "postgresql://test:password@localhost:5432/db"

var lotEndsAtOffset int64 = 100

func loadLotEnv() {
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

func setupLotTest(t *testing.T) (*Service, *pgxpool.Pool, *valkey.Client) {
	t.Helper()
	loadLotEnv()

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
		t.Fatalf("failed to create s3 client: %v", err)
	}

	repo := NewRepo(pool)
	service := NewService(repo, pool, s3Client, valkeyClient)

	t.Cleanup(func() {
		pool.Close()
		_ = valkeyClient.Close()
	})

	return service, pool, valkeyClient
}

func createTestUser(t *testing.T, pool *pgxpool.Pool, balance, hold int) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	uid := uuid.New()
	email := fmt.Sprintf("lotuser_%s@example.com", strings.ReplaceAll(uid.String(), "-", ""))
	name := fmt.Sprintf("User_%s", uid.String()[:8])

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, password_hash, balance, hold, role)
		VALUES ($1, $2, $3, 'hashed_pw', $4, $5, 'user')
	`, uid, name, email, balance, hold)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	return uid
}

func createTestItem(t *testing.T, pool *pgxpool.Pool, ownerID uuid.UUID) int64 {
	t.Helper()
	ctx := context.Background()
	s3ID := uuid.New()
	s3Hash := fmt.Sprintf("hash_%s", s3ID.String())

	_, err := pool.Exec(ctx, "INSERT INTO s3 (id, hash) VALUES ($1, $2)", s3ID, s3Hash)
	if err != nil {
		t.Fatalf("failed to insert s3 record: %v", err)
	}

	var itemID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO items (creator_id, owner_id, s3_id, type)
		VALUES ($1, $2, $3, 'image')
		RETURNING id
	`, ownerID, ownerID, s3ID).Scan(&itemID)
	if err != nil {
		t.Fatalf("failed to insert test item: %v", err)
	}
	return itemID
}

func createTestLot(t *testing.T, pool *pgxpool.Pool, sellerID uuid.UUID, itemID int64, status Status, startAmount, minStep int, endsAt *time.Time) int64 {
	t.Helper()
	ctx := context.Background()

	var startedAt *time.Time
	var finishedAt *time.Time
	now := time.Now().Add(-1 * time.Hour)

	if status == "active" {
		startedAt = &now
		if endsAt == nil {
			offset := atomic.AddInt64(&lotEndsAtOffset, 1)
			future := time.Now().Add(time.Duration(offset) * time.Hour)
			endsAt = &future
		}
	} else if status == "finished" {
		startedAt = &now
		finishedAt = &now
		past := time.Now().Add(-10 * time.Minute)
		endsAt = &past
	}

	var lotID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO lots (seller_id, item_id, status, start_amount, min_step, title, description, started_at, ends_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, 'Test Lot', '{}'::jsonb, $6, $7, $8)
		RETURNING id
	`, sellerID, itemID, status, startAmount, minStep, startedAt, endsAt, finishedAt).Scan(&lotID)
	if err != nil {
		t.Fatalf("failed to insert test lot: %v", err)
	}
	return lotID
}

// -----------------------------------------------------------------------------
// 1. bid with amount <= 0 must return error wrapping ErrInvalidReqBody
// -----------------------------------------------------------------------------
func TestBid_AmountNonPositive(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, nil)

	bidderID := createTestUser(t, pool, 1000, 0)

	// amount = 0
	err := service.bid(ctx, lotID, bidderID, 0)
	if err == nil {
		t.Errorf("bid with amount=0 succeeded, expected ErrInvalidReqBody")
	} else if !errors.Is(err, ErrInvalidReqBody) {
		t.Errorf("bid with amount=0 returned %v, expected error wrapping ErrInvalidReqBody", err)
	}

	// amount < 0
	err = service.bid(ctx, lotID, bidderID, -50)
	if err == nil {
		t.Errorf("bid with amount=-50 succeeded, expected ErrInvalidReqBody")
	} else if !errors.Is(err, ErrInvalidReqBody) {
		t.Errorf("bid with amount=-50 returned %v, expected error wrapping ErrInvalidReqBody", err)
	}
}

// -----------------------------------------------------------------------------
// 2. bid by the seller on their own lot must return error wrapping ErrForbidden
// -----------------------------------------------------------------------------
func TestBid_SellerCannotBidOnOwnLot(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, nil)

	err := service.bid(ctx, lotID, sellerID, 150)
	if err == nil {
		t.Errorf("seller bid on own lot succeeded, expected ErrForbidden")
	} else if !errors.Is(err, ErrForbidden) {
		t.Errorf("seller bid on own lot returned %v, expected error wrapping ErrForbidden", err)
	}
}

// -----------------------------------------------------------------------------
// 3. Bidding on a lot that is not 'active' or does not exist must return an error
// -----------------------------------------------------------------------------
func TestBid_LotNotActiveOrNotFound(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	bidderID := createTestUser(t, pool, 1000, 0)

	// Lot does not exist
	err := service.bid(ctx, 999999999, bidderID, 100)
	if err == nil {
		t.Errorf("bid on non-existent lot succeeded, expected error")
	}

	// Lot in non-active statuses
	nonActiveStatuses := []Status{"draft", "pending_review", "approved", "rejected", "finished", "cancelled"}
	for _, status := range nonActiveStatuses {
		sellerID := createTestUser(t, pool, 1000, 0)
		itemID := createTestItem(t, pool, sellerID)
		lotID := createTestLot(t, pool, sellerID, itemID, status, 100, 10, nil)

		err := service.bid(ctx, lotID, bidderID, 100)
		if err == nil {
			t.Errorf("bid on lot in status '%s' succeeded, expected error", status)
		}
	}
}

// -----------------------------------------------------------------------------
// 4. First bid on a lot:
//    - Must be >= lot.StartAmount. If amount < lot.StartAmount, returns ErrAmountSmall.
//    - If valid, sets bid, increases bidder's hold by amount. Bidder's balance unchanged.
// -----------------------------------------------------------------------------
func TestBid_FirstBid(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	startAmount := 100
	lotID := createTestLot(t, pool, sellerID, itemID, "active", startAmount, 10, nil)

	bidderBalance := 500
	bidderID := createTestUser(t, pool, bidderBalance, 0)

	// Bid less than StartAmount
	err := service.bid(ctx, lotID, bidderID, startAmount-1)
	if err == nil {
		t.Errorf("bid < startAmount succeeded, expected ErrAmountSmall")
	} else if !errors.Is(err, ErrAmountSmall) {
		t.Errorf("bid < startAmount returned %v, expected error wrapping ErrAmountSmall", err)
	}

	// Valid first bid equal to StartAmount
	bidAmount := 120
	err = service.bid(ctx, lotID, bidderID, bidAmount)
	if err != nil {
		t.Fatalf("valid first bid failed: %v", err)
	}

	// Invariants check:
	// 1. Bidder's hold increases by bidAmount
	// 2. Bidder's balance remains unchanged
	var dbBalance, dbHold int
	err = pool.QueryRow(ctx, "SELECT balance, hold FROM users WHERE id = $1", bidderID).Scan(&dbBalance, &dbHold)
	if err != nil {
		t.Fatalf("failed to query bidder: %v", err)
	}
	if dbBalance != bidderBalance {
		t.Errorf("bidder balance changed: expected %d, got %d", bidderBalance, dbBalance)
	}
	if dbHold != bidAmount {
		t.Errorf("bidder hold incorrect: expected %d, got %d", bidAmount, dbHold)
	}

	// 3. Bid record created in bids table
	var count int
	var recordedAmount int
	err = pool.QueryRow(ctx, "SELECT count(*), coalesce(max(amount), 0) FROM bids WHERE lot_id = $1 AND user_id = $2", lotID, bidderID).Scan(&count, &recordedAmount)
	if err != nil {
		t.Fatalf("failed to query bids: %v", err)
	}
	if count != 1 || recordedAmount != bidAmount {
		t.Errorf("expected 1 bid with amount %d, got count=%d, amount=%d", bidAmount, count, recordedAmount)
	}
}

// -----------------------------------------------------------------------------
// 5. Subsequent bids:
//    - If amount < lastBid.Amount + lot.MinStep, returns ErrAmountSmall.
//    - If bidder is already the current leading bidder, returns ErrLider.
//    - If valid, new bidder's hold increases by amount, previous leader's hold decreases by previousBid.Amount.
// -----------------------------------------------------------------------------
func TestBid_SubsequentBids(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	startAmount := 100
	minStep := 20
	lotID := createTestLot(t, pool, sellerID, itemID, "active", startAmount, minStep, nil)

	bidder1Balance := 1000
	bidder1ID := createTestUser(t, pool, bidder1Balance, 0)

	bidder2Balance := 1000
	bidder2ID := createTestUser(t, pool, bidder2Balance, 0)

	// Bidder 1 places first bid: 100
	firstBidAmount := 100
	if err := service.bid(ctx, lotID, bidder1ID, firstBidAmount); err != nil {
		t.Fatalf("first bid failed: %v", err)
	}

	// Bidder 1 tries to bid again while being leader -> ErrLider / ErrLeader
	err := service.bid(ctx, lotID, bidder1ID, 150)
	if err == nil {
		t.Errorf("leader re-bid succeeded, expected ErrLider")
	} else if !errors.Is(err, ErrLider) && !errors.Is(err, ErrLeader) {
		t.Errorf("leader re-bid returned %v, expected error wrapping ErrLider/ErrLeader", err)
	}

	// Bidder 2 bids less than lastBid + minStep (100 + 20 = 120): e.g. 110
	err = service.bid(ctx, lotID, bidder2ID, 110)
	if err == nil {
		t.Errorf("bid < lastBid + minStep succeeded, expected ErrAmountSmall")
	} else if !errors.Is(err, ErrAmountSmall) {
		t.Errorf("bid < lastBid + minStep returned %v, expected error wrapping ErrAmountSmall", err)
	}

	// Bidder 2 places valid outbid: 150
	secondBidAmount := 150
	if err := service.bid(ctx, lotID, bidder2ID, secondBidAmount); err != nil {
		t.Fatalf("second bid failed: %v", err)
	}

	// Invariants check:
	// - Bidder 2: hold = 150, balance = 1000
	var b2Balance, b2Hold int
	_ = pool.QueryRow(ctx, "SELECT balance, hold FROM users WHERE id = $1", bidder2ID).Scan(&b2Balance, &b2Hold)
	if b2Balance != bidder2Balance {
		t.Errorf("bidder 2 balance changed: expected %d, got %d", bidder2Balance, b2Balance)
	}
	if b2Hold != secondBidAmount {
		t.Errorf("bidder 2 hold incorrect: expected %d, got %d", secondBidAmount, b2Hold)
	}

	// - Bidder 1 (previous leader): hold refunded (decreased by 100 -> 0), balance = 1000
	var b1Balance, b1Hold int
	_ = pool.QueryRow(ctx, "SELECT balance, hold FROM users WHERE id = $1", bidder1ID).Scan(&b1Balance, &b1Hold)
	if b1Balance != bidder1Balance {
		t.Errorf("bidder 1 balance changed: expected %d, got %d", bidder1Balance, b1Balance)
	}
	if b1Hold != 0 {
		t.Errorf("bidder 1 hold not refunded: expected 0, got %d", b1Hold)
	}
}

// -----------------------------------------------------------------------------
// 6. Balance & Hold checks:
//    - Available balance is balance - hold.
//    - If bidder tries to bid amount > balance - hold, returns ErrInsufficientBalance.
// -----------------------------------------------------------------------------
func TestBid_InsufficientBalance(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, nil)

	// User has balance 200, hold 50 -> available = 150
	bidderID := createTestUser(t, pool, 200, 50)

	// Try to bid 160 (> available 150)
	err := service.bid(ctx, lotID, bidderID, 160)
	if err == nil {
		t.Errorf("bid exceeding available balance succeeded, expected ErrInsufficientBalance")
	} else if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("bid exceeding available balance returned %v, expected error wrapping ErrInsufficientBalance", err)
	}

	// User has balance 50, hold 0 -> tries to bid 100
	bidder2ID := createTestUser(t, pool, 50, 0)
	err = service.bid(ctx, lotID, bidder2ID, 100)
	if err == nil {
		t.Errorf("bid exceeding total balance succeeded, expected ErrInsufficientBalance")
	} else if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("bid exceeding total balance returned %v, expected error wrapping ErrInsufficientBalance", err)
	}
}

// -----------------------------------------------------------------------------
// 7. Finishing lots:
//    Expired active lot without any bids:
//    - Transition lot to 'finished'
//    - winner_id is NULL
//    - finish_amount is NULL
// -----------------------------------------------------------------------------
func TestFinishLot_ExpiredNoBids(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	pastEndsAt := time.Now().Add(-10 * time.Minute)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, &pastEndsAt)

	err := service.finishLot(ctx, lotID)
	if err != nil {
		t.Fatalf("finishLot failed: %v", err)
	}

	var status string
	var winnerID *uuid.UUID
	var finishAmount *int
	var finishedAt *time.Time
	err = pool.QueryRow(ctx, "SELECT status, winner_id, finish_amount, finished_at FROM lots WHERE id = $1", lotID).
		Scan(&status, &winnerID, &finishAmount, &finishedAt)
	if err != nil {
		t.Fatalf("failed to query lot: %v", err)
	}

	if status != "finished" {
		t.Errorf("expected lot status 'finished', got '%s'", status)
	}
	if winnerID != nil {
		t.Errorf("expected winner_id to be NULL, got %v", winnerID)
	}
	if finishAmount != nil {
		t.Errorf("expected finish_amount to be NULL, got %v", finishAmount)
	}
	if finishedAt == nil {
		t.Errorf("expected finished_at to be set")
	}
}

// -----------------------------------------------------------------------------
// 7b. Finishing lots:
//    Expired active lot with bids:
//    - lot.status = 'finished', winner_id = highest bidder, finish_amount = highest bid amount
//    - Winner's balance decreased by winning amount
//    - Winner's hold decreased by winning amount
//    - Seller's balance increased by winning amount
//    - Item ownership (items.owner_id) transferred to winner
//    - history_item created
//    - Two ledger_entries created:
//        winner: type "transfer_to_seller", negative amount (-winning)
//        seller: type "payout_of_winnings", positive amount (+winning)
// -----------------------------------------------------------------------------
func TestFinishLot_ExpiredWithBids(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerInitialBalance := 1000
	sellerID := createTestUser(t, pool, sellerInitialBalance, 0)
	itemID := createTestItem(t, pool, sellerID)

	futureEndsAt := time.Now().Add(1 * time.Hour)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, &futureEndsAt)

	winnerInitialBalance := 1000
	winnerID := createTestUser(t, pool, winnerInitialBalance, 0)

	winningAmount := 250
	if err := service.bid(ctx, lotID, winnerID, winningAmount); err != nil {
		t.Fatalf("winner bid failed: %v", err)
	}

	// Expire the lot
	if _, err := pool.Exec(ctx, "UPDATE lots SET ends_at = now() - interval '15 seconds' WHERE id = $1", lotID); err != nil {
		t.Fatalf("failed to expire lot: %v", err)
	}

	// Finish the lot
	err := service.finishLot(ctx, lotID)
	if err != nil {
		t.Fatalf("finishLot failed: %v", err)
	}

	// 1. Check lot status, winner_id, finish_amount
	var status string
	var dbWinnerID *uuid.UUID
	var dbFinishAmount *int
	err = pool.QueryRow(ctx, "SELECT status, winner_id, finish_amount FROM lots WHERE id = $1", lotID).
		Scan(&status, &dbWinnerID, &dbFinishAmount)
	if err != nil {
		t.Fatalf("failed to query lot: %v", err)
	}

	if status != "finished" {
		t.Errorf("expected lot status 'finished', got '%s'", status)
	}
	if dbWinnerID == nil || *dbWinnerID != winnerID {
		t.Errorf("expected winner_id %s, got %v", winnerID, dbWinnerID)
	}
	if dbFinishAmount == nil || *dbFinishAmount != winningAmount {
		t.Errorf("expected finish_amount %d, got %v", winningAmount, dbFinishAmount)
	}

	// 2. Check Winner balance & hold
	var winnerBal, winnerHold int
	_ = pool.QueryRow(ctx, "SELECT balance, hold FROM users WHERE id = $1", winnerID).Scan(&winnerBal, &winnerHold)
	expectedWinnerBal := winnerInitialBalance - winningAmount
	if winnerBal != expectedWinnerBal {
		t.Errorf("winner balance: expected %d, got %d", expectedWinnerBal, winnerBal)
	}
	if winnerHold != 0 {
		t.Errorf("winner hold: expected 0, got %d", winnerHold)
	}

	// 3. Check Seller balance
	var sellerBal int
	_ = pool.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1", sellerID).Scan(&sellerBal)
	expectedSellerBal := sellerInitialBalance + winningAmount
	if sellerBal != expectedSellerBal {
		t.Errorf("seller balance: expected %d, got %d", expectedSellerBal, sellerBal)
	}

	// 4. Check item ownership transferred to winner
	var currentOwnerID uuid.UUID
	_ = pool.QueryRow(ctx, "SELECT owner_id FROM items WHERE id = $1", itemID).Scan(&currentOwnerID)
	if currentOwnerID != winnerID {
		t.Errorf("item owner: expected winner %s, got %s", winnerID, currentOwnerID)
	}

	// 5. Check history_item created
	var oldUser, newUser uuid.UUID
	err = pool.QueryRow(ctx, "SELECT old_user, new_user FROM history_item WHERE lot_id = $1", lotID).
		Scan(&oldUser, &newUser)
	if err != nil {
		t.Fatalf("failed to query history_item: %v", err)
	}
	if oldUser != sellerID || newUser != winnerID {
		t.Errorf("history_item: expected old=%s, new=%s, got old=%s, new=%s", sellerID, winnerID, oldUser, newUser)
	}

	// 6. Check two ledger_entries
	rows, err := pool.Query(ctx, "SELECT user_id, amount, type FROM ledger_entries WHERE lot_id = $1 ORDER BY id ASC", lotID)
	if err != nil {
		t.Fatalf("failed to query ledger_entries: %v", err)
	}
	defer rows.Close()

	var ledgerRecords []struct {
		UserID uuid.UUID
		Amount int
		Type   string
	}
	for rows.Next() {
		var r struct {
			UserID uuid.UUID
			Amount int
			Type   string
		}
		if err := rows.Scan(&r.UserID, &r.Amount, &r.Type); err != nil {
			t.Fatalf("failed to scan ledger row: %v", err)
		}
		ledgerRecords = append(ledgerRecords, r)
	}

	if len(ledgerRecords) != 2 {
		t.Errorf("expected 2 ledger_entries, got %d", len(ledgerRecords))
	} else {
		// Winner entry: transfer_to_seller, -winningAmount
		// Seller entry: payout_of_winnings, +winningAmount
		hasWinnerEntry := false
		hasSellerEntry := false
		for _, rec := range ledgerRecords {
			if rec.UserID == winnerID && rec.Amount == -winningAmount && rec.Type == "transfer_to_seller" {
				hasWinnerEntry = true
			}
			if rec.UserID == sellerID && rec.Amount == winningAmount && rec.Type == "payout_of_winnings" {
				hasSellerEntry = true
			}
		}
		if !hasWinnerEntry {
			t.Errorf("missing or invalid winner ledger entry: expected user=%s, amount=%d, type=transfer_to_seller", winnerID, -winningAmount)
		}
		if !hasSellerEntry {
			t.Errorf("missing or invalid seller ledger entry: expected user=%s, amount=%d, type=payout_of_winnings", sellerID, winningAmount)
		}
	}
}

// -----------------------------------------------------------------------------
// 8. Lot moderation (checkLotText & moderateLot):
//    - Title: min 3 runes, max 250 runes.
//    - ShortText: max 500 runes. Text: max 10000 runes.
//    - Prohibited keywords: "казино вулкан", "free money", "купить наркотики", "взлом аккаунтов"
// -----------------------------------------------------------------------------
func TestCheckLotText(t *testing.T) {
	makeLot := func(title, shortText, text string) Lot {
		return Lot{
			Title: title,
			Description: Description{
				ShortText: &shortText,
				Text:      &text,
			},
		}
	}

	// 1. Valid Lot Text
	validLot := makeLot("Great Antique Watch", "Nice summary", "Full detailed description of antique watch.")
	msg, ok := checkLotText(validLot)
	if !ok || msg != "" {
		t.Errorf("expected valid lot to pass moderation (ok=true, msg=''), got ok=%v, msg=%q", ok, msg)
	}

	// 2. Title too short (< 3 runes)
	tooShortLot := makeLot("ab", "short", "text")
	msgShort, okShort := checkLotText(tooShortLot)
	if okShort {
		t.Errorf("expected title < 3 runes to fail moderation, got ok=%v, msg=%q", okShort, msgShort)
	}

	// 3. Title too long (> 250 runes)
	tooLongTitle := makeLot(strings.Repeat("a", 251), "short", "text")
	msgLongTitle, okLongTitle := checkLotText(tooLongTitle)
	if okLongTitle {
		t.Errorf("expected title > 250 runes to fail moderation, got ok=%v, msg=%q", okLongTitle, msgLongTitle)
	}

	// 4. ShortText too long (> 500 runes)
	tooLongShortText := makeLot("Valid Title", strings.Repeat("b", 501), "text")
	msgLongShort, okLongShort := checkLotText(tooLongShortText)
	if okLongShort {
		t.Errorf("expected shortText > 500 runes to fail moderation, got ok=%v, msg=%q", okLongShort, msgLongShort)
	}

	// 5. Text too long (> 10000 runes)
	tooLongText := makeLot("Valid Title", "short", strings.Repeat("c", 10001))
	msgLongText, okLongText := checkLotText(tooLongText)
	if okLongText {
		t.Errorf("expected text > 10000 runes to fail moderation, got ok=%v, msg=%q", okLongText, msgLongText)
	}

	// 6. Prohibited keywords
	prohibitedKeywords := []string{
		"казино вулкан",
		"free money",
		"купить наркотики",
		"взлом аккаунтов",
	}

	for _, kw := range prohibitedKeywords {
		kwLot := makeLot("Title with "+kw, "Short description", "Detailed description")
		_, kwOk := checkLotText(kwLot)
		if kwOk {
			t.Errorf("expected prohibited keyword %q in title to fail moderation", kw)
		}

		kwInTextLot := makeLot("Clean Title", "Clean summary", "Here is "+kw+" for you.")
		_, kwTextOk := checkLotText(kwInTextLot)
		if kwTextOk {
			t.Errorf("expected prohibited keyword %q in text to fail moderation", kw)
		}
	}
}

func TestModerateLot(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)

	// 1. Moderate lot with clean text in pending_review
	cleanLotID := createTestLot(t, pool, sellerID, itemID, "pending_review", 100, 10, nil)
	err := service.moderateLot(ctx, cleanLotID)
	if err != nil {
		t.Fatalf("moderateLot failed for clean lot: %v", err)
	}

	var cleanStatus string
	_ = pool.QueryRow(ctx, "SELECT status FROM lots WHERE id = $1", cleanLotID).Scan(&cleanStatus)
	if cleanStatus != "approved" {
		t.Errorf("expected status 'approved' after moderating clean lot, got '%s'", cleanStatus)
	}

	// 2. Moderate lot with prohibited text in pending_review
	badDescJSON := `{"short_text": "Buy here", "text": "казино вулкан выиграй миллион"}`
	var badLotID int64
	err = pool.QueryRow(ctx, `
		INSERT INTO lots (seller_id, item_id, status, start_amount, min_step, title, description)
		VALUES ($1, $2, 'pending_review', 100, 10, 'Bad Lot', $3::jsonb)
		RETURNING id
	`, sellerID, itemID, badDescJSON).Scan(&badLotID)
	if err != nil {
		t.Fatalf("failed to insert bad lot: %v", err)
	}

	err = service.moderateLot(ctx, badLotID)
	var badStatus string
	_ = pool.QueryRow(ctx, "SELECT status FROM lots WHERE id = $1", badLotID).Scan(&badStatus)
	if badStatus != "rejected" {
		t.Errorf("expected status 'rejected' after moderating prohibited lot, got '%s' (err=%v)", badStatus, err)
	}
}

// -----------------------------------------------------------------------------
// 9. Lot Activation & Cancellation:
//    - activateLot: lot must be 'approved', seller ID must match
//    - cancelLot: seller can cancel lot
// -----------------------------------------------------------------------------
func TestActivateLot(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	otherUserID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)

	// 1. Lot in 'draft' status -> activateLot should fail with ErrLotNotApproved
	draftLotID := createTestLot(t, pool, sellerID, itemID, "draft", 100, 10, nil)
	now := time.Now()
	offset := atomic.AddInt64(&lotEndsAtOffset, 1)
	future := time.Now().Add(time.Duration(offset) * time.Hour)

	_, err := service.activateLot(ctx, draftLotID, sellerID, &now, &future)
	if err == nil {
		t.Errorf("activateLot on draft lot succeeded, expected ErrLotNotApproved")
	} else if !errors.Is(err, ErrLotNotApproved) {
		t.Errorf("activateLot on draft lot returned %v, expected error wrapping ErrLotNotApproved", err)
	}

	// 2. Lot in 'approved' status -> activateLot succeeds
	approvedLotID := createTestLot(t, pool, sellerID, itemID, "approved", 100, 10, nil)
	activeLot, err := service.activateLot(ctx, approvedLotID, sellerID, &now, &future)
	if err != nil {
		t.Fatalf("activateLot on approved lot failed: %v", err)
	}
	if activeLot.Status != "active" {
		t.Errorf("expected activated lot status 'active', got '%s'", activeLot.Status)
	}

	// 3. Non-seller tries to activate
	approvedLot2ID := createTestLot(t, pool, sellerID, itemID, "approved", 100, 10, nil)
	offset2 := atomic.AddInt64(&lotEndsAtOffset, 1)
	future2 := time.Now().Add(time.Duration(offset2) * time.Hour)
	_, err = service.activateLot(ctx, approvedLot2ID, otherUserID, &now, &future2)
	if err == nil {
		t.Errorf("activateLot by non-seller succeeded, expected error")
	}
}

func TestCancelLot(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	otherUserID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)

	lotID := createTestLot(t, pool, sellerID, itemID, "approved", 100, 10, nil)

	// Non-seller cannot cancel lot
	err := service.cancelLot(ctx, lotID, otherUserID)
	if err == nil {
		t.Errorf("cancelLot by non-seller succeeded, expected error")
	}

	// Seller can cancel lot
	err = service.cancelLot(ctx, lotID, sellerID)
	if err != nil {
		t.Fatalf("cancelLot by seller failed: %v", err)
	}

	var status string
	_ = pool.QueryRow(ctx, "SELECT status FROM lots WHERE id = $1", lotID).Scan(&status)
	if status != "cancelled" {
		t.Errorf("expected lot status 'cancelled', got '%s'", status)
	}
}

// -----------------------------------------------------------------------------
// 10. UUID parsing (ExtractUUID)
//     - Standard UUID string
//     - 32-char hex
//     - URL path like "https://.../uuid"
//     - rejects invalid strings
// -----------------------------------------------------------------------------
func TestExtractUUID(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantOK  bool
		wantVal string
	}{
		{
			name:    "Standard UUID",
			raw:     "550e8400-e29b-41d4-a716-446655440000",
			wantOK:  true,
			wantVal: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:    "32-char hex UUID",
			raw:     "550e8400e29b41d4a716446655440000",
			wantOK:  true,
			wantVal: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:    "URL path with UUID",
			raw:     "https://s3.ru1.storage.beget.cloud/auc-photos/550e8400-e29b-41d4-a716-446655440000",
			wantOK:  true,
			wantVal: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:    "Relative path with UUID",
			raw:     "/images/550e8400-e29b-41d4-a716-446655440000",
			wantOK:  true,
			wantVal: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:   "Empty string",
			raw:    "",
			wantOK: false,
		},
		{
			name:   "Random short text",
			raw:    "not-a-uuid",
			wantOK: false,
		},
		{
			name:   "Invalid characters with uuid length",
			raw:    "zzze8400-e29b-41d4-a716-446655440000",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ExtractUUID(tc.raw)
			if ok != tc.wantOK {
				t.Errorf("ExtractUUID(%q): ok = %v, wantOK = %v", tc.raw, ok, tc.wantOK)
			}
			if tc.wantOK && tc.wantVal != "" {
				// Normalize hyphens for comparison if needed
				cleanGot := strings.ReplaceAll(got, "-", "")
				cleanWant := strings.ReplaceAll(tc.wantVal, "-", "")
				if cleanGot != cleanWant {
					t.Errorf("ExtractUUID(%q): got %q, want %q", tc.raw, got, tc.wantVal)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// 11. Description JSON & Images
// -----------------------------------------------------------------------------
// -----------------------------------------------------------------------------
// 11. Description JSON & Images:
//     - UnmarshalJSON properly extracts images and converts to public URLs
//     - GetImages returns slice of image identifiers
//     - MarshalJSON properly serializes description
// -----------------------------------------------------------------------------
func TestDescription_JSON(t *testing.T) {
	loadLotEnv()

	uuid1 := "550e8400-e29b-41d4-a716-446655440000"
	uuid2 := "550e8400-e29b-41d4-a716-446655440001"
	rawJSON := fmt.Sprintf(`{"text":"Vintage Rolex","short_text":"Rolex 1968","images":[%q,%q]}`, uuid1, uuid2)

	var desc Description
	if err := json.Unmarshal([]byte(rawJSON), &desc); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// 1. Check text & shortText
	if desc.Text == nil || *desc.Text != "Vintage Rolex" {
		t.Errorf("expected text 'Vintage Rolex', got %v", desc.Text)
	}
	if desc.ShortText == nil || *desc.ShortText != "Rolex 1968" {
		t.Errorf("expected short_text 'Rolex 1968', got %v", desc.ShortText)
	}

	// 2. Check GetImages()
	images := desc.GetImages()
	if len(images) != 2 {
		t.Fatalf("expected 2 images from GetImages(), got %d: %v", len(images), images)
	}
	if images[0] != uuid1 || images[1] != uuid2 {
		t.Errorf("GetImages mismatch: expected [%s, %s], got %v", uuid1, uuid2, images)
	}

	// 3. Check PhotoURL converted to public URLs containing the image UUIDs
	if desc.PhotoURL == nil {
		t.Fatalf("expected PhotoURL to be populated with public URLs, got nil")
	}
	if !strings.Contains(desc.PhotoURL[0], uuid1) || !strings.Contains(desc.PhotoURL[1], uuid2) {
		t.Errorf("expected PhotoURLs to contain UUIDs, got: %v", desc.PhotoURL)
	}

	// 4. Check MarshalJSON
	marshaled, err := json.Marshal(desc)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	if len(marshaled) == 0 {
		t.Errorf("MarshalJSON returned empty bytes")
	}
}

// -----------------------------------------------------------------------------
// 12. Edge Case: Winner exact balance finishLot
// -----------------------------------------------------------------------------
func TestFinishLot_WinnerExactBalance(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)

	futureEndsAt := time.Now().Add(1 * time.Hour)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, &futureEndsAt)

	// Winner has balance = 250, hold = 0
	winnerID := createTestUser(t, pool, 250, 0)

	// Winner bids exactly 250 (available balance becomes 0: balance = 250, hold = 250)
	if err := service.bid(ctx, lotID, winnerID, 250); err != nil {
		t.Fatalf("winner bid failed: %v", err)
	}

	// Verify balance = 250, hold = 250
	var bal, hold int
	if err := pool.QueryRow(ctx, "SELECT balance, hold FROM users WHERE id = $1", winnerID).Scan(&bal, &hold); err != nil {
		t.Fatalf("failed to query winner: %v", err)
	}
	if bal != 250 || hold != 250 {
		t.Fatalf("precondition failed: balance=%d, hold=%d, expected 250, 250", bal, hold)
	}

	// Expire the lot (ends_at in the past). Call finishLot
	if _, err := pool.Exec(ctx, "UPDATE lots SET ends_at = now() - interval '15 seconds' WHERE id = $1", lotID); err != nil {
		t.Fatalf("failed to expire lot: %v", err)
	}
	err := service.finishLot(ctx, lotID)
	if err != nil {
		t.Errorf("finishLot failed due to balance vs hold deduction ordering: %v", err)
	} else {
		t.Logf("finishLot succeeded for winner with exact balance")
	}
}

// -----------------------------------------------------------------------------
// 13. Edge Case: Bidding on expired active lot
// -----------------------------------------------------------------------------
func TestBid_ExpiredActiveLot(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)

	// Active lot whose ends_at is 5 minutes in the past (already expired, background finish hasn't run yet)
	pastEndsAt := time.Now().Add(-5 * time.Minute - time.Duration(atomic.AddInt64(&lotEndsAtOffset, 1))*time.Second)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, &pastEndsAt)

	bidderID := createTestUser(t, pool, 1000, 0)

	// A bidder tries to place a bid on this lot
	err := service.bid(ctx, lotID, bidderID, 150)
	if err == nil {
		t.Errorf("bid on expired active lot was incorrectly accepted (expected bid to be rejected)")
	} else {
		t.Logf("bid on expired active lot was rejected as expected: %v", err)
	}
}

// -----------------------------------------------------------------------------
// 14. Edge Case: Activate lots with same ends_at timestamp
// -----------------------------------------------------------------------------
func TestActivateLot_SameEndsAtCollision(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	seller1ID := createTestUser(t, pool, 1000, 0)
	item1ID := createTestItem(t, pool, seller1ID)
	lot1ID := createTestLot(t, pool, seller1ID, item1ID, "approved", 100, 10, nil)

	seller2ID := createTestUser(t, pool, 1000, 0)
	item2ID := createTestItem(t, pool, seller2ID)
	lot2ID := createTestLot(t, pool, seller2ID, item2ID, "approved", 100, 10, nil)

	now := time.Now()
	offset := atomic.AddInt64(&lotEndsAtOffset, 100)
	sameEndsAt := time.Now().Add(time.Duration(offset) * time.Hour).Truncate(time.Second)

	// Try to activate first lot
	_, err1 := service.activateLot(ctx, lot1ID, seller1ID, &now, &sameEndsAt)
	if err1 != nil {
		t.Fatalf("first activateLot failed: %v", err1)
	}

	// Try to activate second lot with the EXACT SAME ends_at timestamp
	_, err2 := service.activateLot(ctx, lot2ID, seller2ID, &now, &sameEndsAt)
	if err2 != nil {
		t.Errorf("database index collision occurred on ux_lot_item_active: %v", err2)
	} else {
		t.Logf("both lots successfully activated at the same ends_at")
	}
}

// -----------------------------------------------------------------------------
// 15. AdminCancelLot: Active lot with bids
// -----------------------------------------------------------------------------
func TestAdminCancelLot_ActiveWithBids(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	futureEndsAt := time.Now().Add(1 * time.Hour)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, &futureEndsAt)

	bidderID := createTestUser(t, pool, 500, 0)
	bidAmount := 150
	if err := service.bid(ctx, lotID, bidderID, bidAmount); err != nil {
		t.Fatalf("bid failed: %v", err)
	}

	// Verify bidder has hold = 150
	var preHold int
	if err := pool.QueryRow(ctx, "SELECT hold FROM users WHERE id = $1", bidderID).Scan(&preHold); err != nil {
		t.Fatalf("failed to query bidder hold: %v", err)
	}
	if preHold != bidAmount {
		t.Fatalf("expected pre-hold %d, got %d", bidAmount, preHold)
	}

	// Admin calls AdminCancelLot
	if err := service.AdminCancelLot(ctx, lotID); err != nil {
		t.Fatalf("AdminCancelLot failed: %v", err)
	}

	// Assert: lot status becomes cancelled
	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM lots WHERE id = $1", lotID).Scan(&status); err != nil {
		t.Fatalf("failed to query lot status: %v", err)
	}
	if status != "cancelled" {
		t.Errorf("expected lot status 'cancelled', got '%s'", status)
	}

	// Assert: leading bidder's hold is refunded (decreased by bid amount back to 0)
	var postBal, postHold int
	if err := pool.QueryRow(ctx, "SELECT balance, hold FROM users WHERE id = $1", bidderID).Scan(&postBal, &postHold); err != nil {
		t.Fatalf("failed to query bidder after cancel: %v", err)
	}
	if postHold != 0 {
		t.Errorf("expected bidder hold refunded to 0, got %d", postHold)
	}
	if postBal != 500 {
		t.Errorf("expected bidder balance 500, got %d", postBal)
	}
}

// -----------------------------------------------------------------------------
// 16. AdminCancelLot: Non-active statuses
// -----------------------------------------------------------------------------
func TestAdminCancelLot_NonActiveStatuses(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	// Admin can cancel lots in draft, pending_review, approved
	cancellableStatuses := []Status{"draft", "pending_review", "approved"}
	for _, status := range cancellableStatuses {
		sellerID := createTestUser(t, pool, 1000, 0)
		itemID := createTestItem(t, pool, sellerID)
		lotID := createTestLot(t, pool, sellerID, itemID, status, 100, 10, nil)

		if err := service.AdminCancelLot(ctx, lotID); err != nil {
			t.Errorf("AdminCancelLot failed for status %q: %v", status, err)
		} else {
			var dbStatus string
			_ = pool.QueryRow(ctx, "SELECT status FROM lots WHERE id = $1", lotID).Scan(&dbStatus)
			if dbStatus != "cancelled" {
				t.Errorf("expected status 'cancelled' for %q lot, got %q", status, dbStatus)
			}
		}
	}

	// Admin CANNOT cancel lots in finished or already cancelled status (should return error)
	nonCancellableStatuses := []Status{"finished", "cancelled"}
	for _, status := range nonCancellableStatuses {
		sellerID := createTestUser(t, pool, 1000, 0)
		itemID := createTestItem(t, pool, sellerID)
		lotID := createTestLot(t, pool, sellerID, itemID, status, 100, 10, nil)

		if err := service.AdminCancelLot(ctx, lotID); err == nil {
			t.Errorf("AdminCancelLot succeeded for status %q, expected error", status)
		} else {
			t.Logf("AdminCancelLot correctly failed for status %q: %v", status, err)
		}
	}
}

// -----------------------------------------------------------------------------
// 17. cancelLot: Seller cancels lot in pending_review
// -----------------------------------------------------------------------------
func TestCancelLot_SellerPendingReview(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	lotID := createTestLot(t, pool, sellerID, itemID, "pending_review", 100, 10, nil)

	err := service.cancelLot(ctx, lotID, sellerID)
	var dbStatus string
	_ = pool.QueryRow(ctx, "SELECT status FROM lots WHERE id = $1", lotID).Scan(&dbStatus)

	if err != nil {
		t.Logf("Seller cannot cancel lot in pending_review: %v (lot status: %s)", err, dbStatus)
		if dbStatus != "pending_review" {
			t.Errorf("expected status to remain 'pending_review', got %q", dbStatus)
		}
	} else {
		t.Logf("Seller successfully cancelled lot in pending_review")
		if dbStatus != "cancelled" {
			t.Errorf("expected status 'cancelled', got %q", dbStatus)
		}
	}
}

// -----------------------------------------------------------------------------
// 18. bid: Future scheduled lot
// -----------------------------------------------------------------------------
func TestBid_FutureScheduledLot(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)

	futureStart := time.Now().Add(2 * time.Hour)
	futureEnd := time.Now().Add(24 * time.Hour)

	var lotID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO lots (seller_id, item_id, status, start_amount, min_step, title, description, started_at, ends_at)
		VALUES ($1, $2, 'active', 100, 10, 'Future Lot', '{}'::jsonb, $3, $4)
		RETURNING id
	`, sellerID, itemID, futureStart, futureEnd).Scan(&lotID)
	if err != nil {
		t.Fatalf("failed to insert future scheduled lot: %v", err)
	}

	bidderID := createTestUser(t, pool, 1000, 0)
	err = service.bid(ctx, lotID, bidderID, 100)
	if err == nil {
		t.Errorf("bid on future scheduled lot succeeded before start time; expected rejection")
	} else {
		t.Logf("bid on future scheduled lot rejected as expected: %v", err)
	}
}

// -----------------------------------------------------------------------------
// 19. bid: Deleted previous bidder
// -----------------------------------------------------------------------------
func TestBid_DeletedPreviousBidder(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	futureEnd := time.Now().Add(2 * time.Hour)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, &futureEnd)

	user1ID := createTestUser(t, pool, 1000, 0)
	user2ID := createTestUser(t, pool, 1000, 0)

	// User 1 places bid of 100
	if err := service.bid(ctx, lotID, user1ID, 100); err != nil {
		t.Fatalf("user 1 bid failed: %v", err)
	}

	// User 1 is soft-deleted
	_, err := pool.Exec(ctx, "UPDATE users SET deleted_at = now() WHERE id = $1", user1ID)
	if err != nil {
		t.Fatalf("failed to soft-delete user 1: %v", err)
	}

	// User 2 attempts to place an outbid of 120
	err = service.bid(ctx, lotID, user2ID, 120)
	if err != nil {
		t.Errorf("user 2 outbid failed when previous bidder is deleted: %v", err)
	} else {
		t.Logf("user 2 outbid succeeded when previous bidder is deleted")
		var u2Hold int
		_ = pool.QueryRow(ctx, "SELECT hold FROM users WHERE id = $1", user2ID).Scan(&u2Hold)
		if u2Hold != 120 {
			t.Errorf("expected user 2 hold 120, got %d", u2Hold)
		}
	}
}

// -----------------------------------------------------------------------------
// 20. finishLot: Deleted winner
// -----------------------------------------------------------------------------
func TestFinishLot_DeletedWinner(t *testing.T) {
	service, pool, _ := setupLotTest(t)
	ctx := context.Background()

	sellerID := createTestUser(t, pool, 1000, 0)
	itemID := createTestItem(t, pool, sellerID)
	futureEnd := time.Now().Add(2 * time.Hour)
	lotID := createTestLot(t, pool, sellerID, itemID, "active", 100, 10, &futureEnd)

	user1ID := createTestUser(t, pool, 1000, 0)
	winningAmount := 150
	if err := service.bid(ctx, lotID, user1ID, winningAmount); err != nil {
		t.Fatalf("user 1 bid failed: %v", err)
	}

	// User 1 is soft-deleted
	_, err := pool.Exec(ctx, "UPDATE users SET deleted_at = now() WHERE id = $1", user1ID)
	if err != nil {
		t.Fatalf("failed to soft-delete user 1: %v", err)
	}

	// Expire lot
	_, err = pool.Exec(ctx, "UPDATE lots SET ends_at = now() - interval '15 seconds' WHERE id = $1", lotID)
	if err != nil {
		t.Fatalf("failed to expire lot: %v", err)
	}

	// Finish lot
	err = service.finishLot(ctx, lotID)
	if err != nil {
		t.Errorf("finishLot failed for deleted winner: %v", err)
	} else {
		t.Logf("finishLot succeeded for deleted winner")
	}
}

