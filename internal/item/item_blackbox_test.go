package item

import (
	"bufio"
	"context"
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

func loadItemEnv() {
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

func setupItemTest(t *testing.T) (*Service, *pgxpool.Pool, *valkey.Client) {
	t.Helper()
	loadItemEnv()

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
	service := NewService(pool, s3Client, repo, valkeyClient)

	t.Cleanup(func() {
		pool.Close()
		_ = valkeyClient.Close()
	})

	return service, pool, valkeyClient
}

func createTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	uid := uuid.New()
	email := fmt.Sprintf("itemuser_%s@example.com", strings.ReplaceAll(uid.String(), "-", ""))
	name := fmt.Sprintf("User_%s", uid.String()[:8])

	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, password_hash, balance, hold, role)
		VALUES ($1, $2, $3, 'hashed_pw', 1000, 0, 'user')
	`, uid, name, email)
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

func createTestLot(t *testing.T, pool *pgxpool.Pool, sellerID uuid.UUID, itemID int64, status string) int64 {
	t.Helper()
	ctx := context.Background()

	var startedAt *time.Time
	var endsAt *time.Time
	now := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	if status == "active" {
		startedAt = &now
		endsAt = &future
	}

	var lotID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO lots (seller_id, item_id, status, start_amount, min_step, title, description, started_at, ends_at)
		VALUES ($1, $2, $3, 100, 10, 'Test Lot', '{}'::jsonb, $4, $5)
		RETURNING id
	`, sellerID, itemID, status, startedAt, endsAt).Scan(&lotID)
	if err != nil {
		t.Fatalf("failed to insert test lot: %v", err)
	}
	return lotID
}

// -----------------------------------------------------------------------------
// 1. AdminTransferItem: Success
// -----------------------------------------------------------------------------
func TestAdminTransferItem_Success(t *testing.T) {
	service, pool, _ := setupItemTest(t)
	ctx := context.Background()

	userA := createTestUser(t, pool)
	userB := createTestUser(t, pool)
	itemID := createTestItem(t, pool, userA)

	desc := "Admin reassign"
	err := service.AdminTransferItem(ctx, itemID, userB, desc)
	if err != nil {
		t.Fatalf("AdminTransferItem failed: %v", err)
	}

	// Assert: items.owner_id is now User B
	var ownerID uuid.UUID
	err = pool.QueryRow(ctx, "SELECT owner_id FROM items WHERE id = $1", itemID).Scan(&ownerID)
	if err != nil {
		t.Fatalf("failed to query item owner: %v", err)
	}
	if ownerID != userB {
		t.Errorf("expected owner %s, got %s", userB, ownerID)
	}

	// Assert: record in history_item exists with item_id, old_user = A, new_user = B, description = "Admin reassign"
	var oldUser, newUser uuid.UUID
	var dbDesc *string
	err = pool.QueryRow(ctx, `
		SELECT old_user, new_user, description 
		FROM history_item 
		WHERE item_id = $1 
		ORDER BY id DESC LIMIT 1
	`, itemID).Scan(&oldUser, &newUser, &dbDesc)
	if err != nil {
		t.Fatalf("failed to query history_item: %v", err)
	}
	if oldUser != userA || newUser != userB {
		t.Errorf("expected old=%s, new=%s, got old=%s, new=%s", userA, userB, oldUser, newUser)
	}
	if dbDesc == nil || *dbDesc != desc {
		t.Errorf("expected description %q, got %v", desc, dbDesc)
	}
}

// -----------------------------------------------------------------------------
// 2. AdminTransferItem: Same Owner
// -----------------------------------------------------------------------------
func TestAdminTransferItem_SameOwner(t *testing.T) {
	service, pool, _ := setupItemTest(t)
	ctx := context.Background()

	userA := createTestUser(t, pool)
	itemID := createTestItem(t, pool, userA)

	err := service.AdminTransferItem(ctx, itemID, userA, "Transfer to self")
	if err == nil {
		t.Errorf("AdminTransferItem to current owner succeeded, expected error")
	} else {
		t.Logf("AdminTransferItem to current owner correctly returned error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// 3. AdminTransferItem: Active Lot
// -----------------------------------------------------------------------------
func TestAdminTransferItem_ActiveLot(t *testing.T) {
	service, pool, _ := setupItemTest(t)
	ctx := context.Background()

	userA := createTestUser(t, pool)
	userB := createTestUser(t, pool)
	itemID := createTestItem(t, pool, userA)
	createTestLot(t, pool, userA, itemID, "active")

	err := service.AdminTransferItem(ctx, itemID, userB, "Transfer active lot item")
	if err == nil {
		t.Errorf("AdminTransferItem succeeded on item listed in active lot; expected error")
	} else {
		t.Logf("AdminTransferItem correctly blocked item in active lot: %v", err)
	}
}

// -----------------------------------------------------------------------------
// 4. AdminTransferItem: Approved Lot
// -----------------------------------------------------------------------------
func TestAdminTransferItem_ApprovedLot(t *testing.T) {
	service, pool, _ := setupItemTest(t)
	ctx := context.Background()

	userA := createTestUser(t, pool)
	userB := createTestUser(t, pool)
	itemID := createTestItem(t, pool, userA)
	createTestLot(t, pool, userA, itemID, "approved")

	err := service.AdminTransferItem(ctx, itemID, userB, "Transfer approved lot item")
	if err != nil {
		t.Logf("AdminTransferItem blocked transferring item in approved lot: %v", err)
	} else {
		t.Errorf("AdminTransferItem allowed transferring item with approved lot (expected it to be blocked)")
	}
}

// -----------------------------------------------------------------------------
// 5. AdminTransferItem: Invalid Params
// -----------------------------------------------------------------------------
func TestAdminTransferItem_InvalidParams(t *testing.T) {
	service, pool, _ := setupItemTest(t)
	ctx := context.Background()

	userA := createTestUser(t, pool)
	itemID := createTestItem(t, pool, userA)

	// itemID <= 0
	if err := service.AdminTransferItem(ctx, 0, userA, "desc"); err == nil {
		t.Errorf("AdminTransferItem with itemID=0 succeeded, expected error")
	}
	if err := service.AdminTransferItem(ctx, -1, userA, "desc"); err == nil {
		t.Errorf("AdminTransferItem with itemID=-1 succeeded, expected error")
	}

	// newOwner == uuid.Nil()
	if err := service.AdminTransferItem(ctx, itemID, uuid.Nil(), "desc"); err == nil {
		t.Errorf("AdminTransferItem with newOwner=uuid.Nil() succeeded, expected error")
	}

	// Non-existent item ID
	if err := service.AdminTransferItem(ctx, 999999999, userA, "desc"); err == nil {
		t.Errorf("AdminTransferItem with non-existent itemID succeeded, expected error")
	}
}

// -----------------------------------------------------------------------------
// 6. GiftItem: Active Lot
// -----------------------------------------------------------------------------
func TestGiftItem_ActiveLot(t *testing.T) {
	service, pool, _ := setupItemTest(t)
	ctx := context.Background()

	userA := createTestUser(t, pool)
	userB := createTestUser(t, pool)
	itemID := createTestItem(t, pool, userA)
	createTestLot(t, pool, userA, itemID, "active")

	err := service.giftItem(ctx, userA, userB, itemID, "Gift during active auction")
	if err != nil {
		t.Logf("giftItem blocked transferring item in active lot as expected: %v", err)
	} else {
		t.Errorf("giftItem allowed transferring item in active auction lot (expected to block)")
	}
}
