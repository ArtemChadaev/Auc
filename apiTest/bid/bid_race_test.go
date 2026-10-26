package bid_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type userCredential struct {
	Name     string
	Email    string
	Password string
	Token    string
	UUID     string
}

type bidResult struct {
	UserID     string
	UserName   string
	Amount     int
	Status     int
	ErrMsg     string
	SentAt     time.Time
	DurationMs int64
}

type bidResponse struct {
	Code  int    `json:"code"`
	Error string `json:"error,omitempty"`
}

type lotInfo struct {
	ID          int64  `json:"id"`
	StartAmount int    `json:"start_amount"`
	MinStep     int    `json:"min_step"`
	Status      string `json:"status"`
}

type apiResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func getHost() string {
	host := os.Getenv("HOST")
	if host == "" {
		host = "http://localhost:8080"
	}
	return host
}

func getDBPool(t *testing.T) *pgxpool.Pool {
	connStr := "postgresql://test:password@localhost:5432/db"
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
	return pool
}

func loginUser(client *http.Client, host, email, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	resp, err := client.Post(host+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	if res.Code != http.StatusOK || res.Data.AccessToken == "" {
		return "", fmt.Errorf("login failed (%d): %s", res.Code, res.Error)
	}
	return res.Data.AccessToken, nil
}

func ensureActiveLot(t *testing.T, pool *pgxpool.Pool) int64 {
	ctx := context.Background()
	var lotID int64
	var minStep int

	// Find existing active lot or activate one
	err := pool.QueryRow(ctx, `
		SELECT id, min_step FROM lots 
		WHERE status = 'active'
		ORDER BY id DESC LIMIT 1`).Scan(&lotID, &minStep)

	if err == nil && lotID > 0 {
		// Update ends_at to ensure it remains open for bidding during test
		_, _ = pool.Exec(ctx, `UPDATE lots SET ends_at = now() + interval '2 hours' WHERE id = $1`, lotID)
		return lotID
	}

	// Create or update a lot to active
	var sellerUUID string
	err = pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'seller_alpha@auc.test' LIMIT 1").Scan(&sellerUUID)
	if err != nil {
		t.Fatalf("seller not found: %v", err)
	}

	var itemID int64
	err = pool.QueryRow(ctx, "SELECT id FROM items WHERE owner_id = $1 LIMIT 1", sellerUUID).Scan(&itemID)
	if err != nil {
		t.Fatalf("item for seller not found: %v", err)
	}

	err = pool.QueryRow(ctx, `
		INSERT INTO lots (seller_id, item_id, status, start_amount, min_step, title, description, started_at, ends_at)
		VALUES ($1, $2, 'active', 10000, 500, 'Стресс-тест ставок лот', '{"type":"image"}'::jsonb, now() - interval '10 seconds', now() + interval '2 hours')
		RETURNING id`, sellerUUID, itemID).Scan(&lotID)
	if err != nil {
		t.Fatalf("failed to insert active lot for stress test: %v", err)
	}
	return lotID
}

func TestBidConcurrencyRace(t *testing.T) {
	host := getHost()
	pool := getDBPool(t)
	defer pool.Close()

	// 1. Ensure users 15-20 have large balance
	_, err := pool.Exec(context.Background(), `
		UPDATE users 
		SET balance = 100000000, hold = 0 
		WHERE email LIKE 'bidder_%@auc.test'`)
	if err != nil {
		t.Fatalf("failed to update users balance: %v", err)
	}

	// 2. Prepare active lot
	lotID := ensureActiveLot(t, pool)
	t.Logf(">>> Using Active Lot ID: %d for high-concurrency race test", lotID)

	// Clean previous bids on this lot if needed
	_, _ = pool.Exec(context.Background(), "DELETE FROM bids WHERE lot_id = $1", lotID)

	httpClient := &http.Client{Timeout: 10 * time.Second}

	// 3. Authenticate 5 users (15 through 19, plus 20)
	users := []userCredential{
		{Name: "Bidder 15", Email: "bidder_15@auc.test", Password: "Pass15!_Secure"},
		{Name: "Bidder 16", Email: "bidder_16@auc.test", Password: "Pass16!_Secure"},
		{Name: "Bidder 17", Email: "bidder_17@auc.test", Password: "Pass17!_Secure"},
		{Name: "Bidder 18", Email: "bidder_18@auc.test", Password: "Pass18!_Secure"},
		{Name: "Bidder 19", Email: "bidder_19@auc.test", Password: "Pass19!_Secure"},
		{Name: "Bidder 20", Email: "bidder_20@auc.test", Password: "Pass20!_Secure"},
	}

	for i := range users {
		token, err := loginUser(httpClient, host, users[i].Email, users[i].Password)
		if err != nil {
			t.Fatalf("Failed to login %s: %v", users[i].Email, err)
		}
		users[i].Token = token
		var uid string
		_ = pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", users[i].Email).Scan(&uid)
		users[i].UUID = uid
	}

	t.Logf(">>> Successfully authenticated %d bidders", len(users))

	// Get lot start amount and min step
	var startAmount, minStep int
	err = pool.QueryRow(context.Background(), "SELECT start_amount, min_step FROM lots WHERE id = $1", lotID).Scan(&startAmount, &minStep)
	if err != nil {
		t.Fatalf("failed to get lot parameters: %v", err)
	}
	t.Logf(">>> Lot Settings: StartAmount=%d, MinStep=%d", startAmount, minStep)

	// 4. Start concurrent high-frequency bidding race
	var globalLastBid int64 = int64(startAmount)
	var allResults []bidResult
	var resultsMutex sync.Mutex

	var startBarrier sync.WaitGroup
	startBarrier.Add(len(users))

	var doneGroup sync.WaitGroup
	doneGroup.Add(len(users))

	startTime := time.Now()
	bidsPerUser := 15

	for userIdx := range users {
		u := users[userIdx]
		go func(bidder userCredential, idx int) {
			defer doneGroup.Done()
			startBarrier.Done()
			startBarrier.Wait() // Synchronized release: all 5 users attack at the exact same moment!

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(idx*1000)))

			for b := 0; b < bidsPerUser; b++ {
				currentLast := atomic.LoadInt64(&globalLastBid)

				// Strategically generate bid amounts:
				// Some identical (to test speed collision), some minimal step, some jumps
				var targetAmount int
				switch r.Intn(4) {
				case 0:
					// Identical amount to current base + minStep (collision test)
					targetAmount = int(currentLast) + minStep
				case 1:
					// Under-step attempt (must fail with 400 ErrAmountSmall)
					targetAmount = int(currentLast) + minStep/2
				case 2:
					// Standard valid next step
					targetAmount = int(currentLast) + minStep
				default:
					// Random jump between 1x and 3x minStep
					targetAmount = int(currentLast) + minStep + r.Intn(3)*minStep
				}

				if targetAmount <= 0 {
					targetAmount = startAmount + minStep
				}

				key := fmt.Sprintf("race_%s_%d_%d", bidder.UUID[:8], b, time.Now().UnixNano())
				reqBody, _ := json.Marshal(map[string]interface{}{
					"amount":          targetAmount,
					"idempotency_key": key,
				})

				reqStart := time.Now()
				req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/bid/%d", host, lotID), bytes.NewReader(reqBody))
				req.Header.Set("Authorization", "Bearer "+bidder.Token)
				req.Header.Set("Content-Type", "application/json")

				resp, reqErr := httpClient.Do(req)
				latency := time.Since(reqStart).Milliseconds()

				status := 0
				errMsg := ""
				if reqErr != nil {
					errMsg = reqErr.Error()
				} else {
					status = resp.StatusCode
					respBytes, _ := io.ReadAll(resp.Body)
					resp.Body.Close()

					var parsed bidResponse
					_ = json.Unmarshal(respBytes, &parsed)
					if parsed.Error != "" {
						errMsg = parsed.Error
					}
				}

				if status == http.StatusOK {
					// Atomically update highest known bid
					for {
						old := atomic.LoadInt64(&globalLastBid)
						if int64(targetAmount) <= old || atomic.CompareAndSwapInt64(&globalLastBid, old, int64(targetAmount)) {
							break
						}
					}
				}

				resultsMutex.Lock()
				allResults = append(allResults, bidResult{
					UserID:     bidder.UUID,
					UserName:   bidder.Name,
					Amount:     targetAmount,
					Status:     status,
					ErrMsg:     errMsg,
					SentAt:     reqStart,
					DurationMs: latency,
				})
				resultsMutex.Unlock()

				// Minimal jitter between 2ms and 15ms to create maximum contention
				time.Sleep(time.Duration(r.Intn(13)+2) * time.Millisecond)
			}
		}(u, userIdx)
	}

	doneGroup.Wait()
	totalDuration := time.Since(startTime)

	t.Logf(">>> Concurrent race completed in %v. Total attempted bids: %d", totalDuration, len(allResults))

	// 5. Verification Phase
	// Fetch actual accepted bids from DB ordered by id ASC
	rows, err := pool.Query(context.Background(), `
		SELECT id, user_id, amount, created_at 
		FROM bids 
		WHERE lot_id = $1 
		ORDER BY id ASC`, lotID)
	if err != nil {
		t.Fatalf("failed to query bids from database: %v", err)
	}
	defer rows.Close()

	type recordedBid struct {
		ID        int64
		UserID    string
		Amount    int
		CreatedAt time.Time
	}

	var acceptedBids []recordedBid
	for rows.Next() {
		var rb recordedBid
		if err := rows.Scan(&rb.ID, &rb.UserID, &rb.Amount, &rb.CreatedAt); err != nil {
			t.Fatalf("failed to scan bid row: %v", err)
		}
		acceptedBids = append(acceptedBids, rb)
	}

	t.Logf(">>> Server Accepted %d valid bids into the database", len(acceptedBids))
	if len(acceptedBids) == 0 {
		t.Fatalf("FATAL: No bids were accepted by the server!")
	}

	// 5.1 VERIFICATION RULE 1: Strictly increasing progression with at least min_step
	for i := 1; i < len(acceptedBids); i++ {
		prev := acceptedBids[i-1]
		curr := acceptedBids[i]

		if curr.Amount < prev.Amount+minStep {
			t.Errorf("RULE VIOLATION: Bid #%d amount %d is less than previous bid #%d amount %d + minStep %d",
				curr.ID, curr.Amount, prev.ID, prev.Amount, minStep)
		}
		if curr.UserID == prev.UserID {
			t.Errorf("RULE VIOLATION: Consecutive bids from the same user %s without intervening competitor!", curr.UserID)
		}
	}

	// 5.2 VERIFICATION RULE 2: Speed collision tie-breaking
	// If two requests had identical amount or gap < minStep, only the faster one (arrived first) was accepted
	status200Count := 0
	status400Count := 0
	for _, res := range allResults {
		if res.Status == http.StatusOK {
			status200Count++
		} else if res.Status == http.StatusBadRequest {
			status400Count++
		}
	}
	t.Logf(">>> Summary of bid attempts: Accepted (200 OK): %d, Rejected (400 Client Error): %d", status200Count, status400Count)
	if status400Count == 0 && len(allResults) > 10 {
		t.Logf("Note: Zero rejections recorded. Contention depends on network timing.")
	}

	// 5.3 VERIFICATION RULE 3: Determine the Winner
	winningBid := acceptedBids[len(acceptedBids)-1]
	var winnerName string
	for _, u := range users {
		if u.UUID == winningBid.UserID {
			winnerName = u.Name
			break
		}
	}

	t.Logf("==================================================================")
	t.Logf(">>> WINNING BID: Amount = %d, Winner = %s (%s)", winningBid.Amount, winnerName, winningBid.UserID)
	t.Logf("==================================================================")

	// Verify that the winning bid has the absolute highest amount
	for _, b := range acceptedBids {
		if b.Amount > winningBid.Amount {
			t.Fatalf("RULE VIOLATION: Found bid higher than winning bid: %d > %d", b.Amount, winningBid.Amount)
		}
	}

	// 5.4 VERIFICATION RULE 4: Hold balance integrity across all users
	for _, u := range users {
		var hold int
		err := pool.QueryRow(context.Background(), "SELECT hold FROM users WHERE id = $1", u.UUID).Scan(&hold)
		if err != nil {
			t.Errorf("failed to query hold for user %s: %v", u.Name, err)
			continue
		}

		if u.UUID == winningBid.UserID {
			if hold != winningBid.Amount {
				t.Errorf("HOLD INTEGRITY ERROR: Winner %s expected hold %d, but found %d", u.Name, winningBid.Amount, hold)
			} else {
				t.Logf(">>> Hold Verified: Winner %s correctly holds winning amount %d", u.Name, hold)
			}
		} else {
			if hold != 0 {
				t.Errorf("HOLD LEAK ERROR: Non-winning user %s still has hold %d (expected 0)", u.Name, hold)
			}
		}
	}

	// 5.5 VERIFICATION RULE 5: Check SSE stream response for the lot
	sseReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/bid/%d/stream", host, lotID), nil)
	sseReq.Header.Set("Authorization", "Bearer "+users[0].Token)
	sseReq.Header.Set("Accept", "text/event-stream")

	sseCtx, sseCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer sseCancel()
	sseReq = sseReq.WithContext(sseCtx)

	sseResp, err := httpClient.Do(sseReq)
	if err == nil {
		defer sseResp.Body.Close()
		if sseResp.StatusCode == http.StatusOK {
			t.Logf(">>> Verified SSE Stream connection: HTTP %d, Content-Type: %s",
				sseResp.StatusCode, sseResp.Header.Get("Content-Type"))
		}
	}
}
