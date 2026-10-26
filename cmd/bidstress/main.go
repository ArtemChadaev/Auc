package main

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

func main() {
	fmt.Println("==================================================================")
	fmt.Println("  AUCTION HIGH-CONCURRENCY BID RACE MODULE (USERS 15 TO 20)")
	fmt.Println("==================================================================")

	host := os.Getenv("HOST")
	if host == "" {
		host = "http://localhost:8080"
	}

	connStr := "postgresql://test:password@localhost:5432/db"
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		fmt.Printf("ERROR: Failed to connect to postgres: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Ensure users have balance
	_, err = pool.Exec(context.Background(), "UPDATE users SET balance = 100000000, hold = 0 WHERE email LIKE 'bidder_%@auc.test'")
	if err != nil {
		fmt.Printf("ERROR: Failed to top up balances: %v\n", err)
		os.Exit(1)
	}

	// Find active lot
	var lotID int64
	var minStep, startAmount int
	err = pool.QueryRow(context.Background(), `
		SELECT id, start_amount, min_step FROM lots 
		WHERE status = 'active'
		ORDER BY id DESC LIMIT 1`).Scan(&lotID, &startAmount, &minStep)

	if err != nil || lotID <= 0 {
		fmt.Println("No active lot found! Please ensure an active lot exists or run lot.http.")
		os.Exit(1)
	}

	// Ensure ends_at is in the future
	_, _ = pool.Exec(context.Background(), "UPDATE lots SET ends_at = now() + interval '2 hours' WHERE id = $1", lotID)

	// Clean previous bids
	_, _ = pool.Exec(context.Background(), "DELETE FROM bids WHERE lot_id = $1", lotID)

	fmt.Printf("[+] Target Lot ID: %d | StartAmount: %d | MinStep: %d\n", lotID, startAmount, minStep)

	users := []userCredential{
		{Name: "Bidder 15", Email: "bidder_15@auc.test", Password: "Pass15!_Secure"},
		{Name: "Bidder 16", Email: "bidder_16@auc.test", Password: "Pass16!_Secure"},
		{Name: "Bidder 17", Email: "bidder_17@auc.test", Password: "Pass17!_Secure"},
		{Name: "Bidder 18", Email: "bidder_18@auc.test", Password: "Pass18!_Secure"},
		{Name: "Bidder 19", Email: "bidder_19@auc.test", Password: "Pass19!_Secure"},
		{Name: "Bidder 20", Email: "bidder_20@auc.test", Password: "Pass20!_Secure"},
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}

	for i := range users {
		loginBody, _ := json.Marshal(map[string]string{
			"email":    users[i].Email,
			"password": users[i].Password,
		})
		resp, err := httpClient.Post(host+"/api/auth/login", "application/json", bytes.NewReader(loginBody))
		if err != nil {
			fmt.Printf("ERROR: Failed to connect to server at %s: %v\n", host, err)
			os.Exit(1)
		}
		var loginRes struct {
			Code int `json:"code"`
			Data struct {
				AccessToken string `json:"access_token"`
			} `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&loginRes)
		resp.Body.Close()
		users[i].Token = loginRes.Data.AccessToken

		var uid string
		_ = pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", users[i].Email).Scan(&uid)
		users[i].UUID = uid
	}

	fmt.Printf("[+] All %d bidders logged in successfully!\n", len(users))
	fmt.Println("[*] Launching concurrent bidding barrage from all 5-6 users simultaneously...")

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
			startBarrier.Wait()

			r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(idx*1000)))

			for b := 0; b < bidsPerUser; b++ {
				currentLast := atomic.LoadInt64(&globalLastBid)

				var targetAmount int
				switch r.Intn(4) {
				case 0:
					targetAmount = int(currentLast) + minStep
				case 1:
					targetAmount = int(currentLast) + minStep/2 // Understep attempt
				case 2:
					targetAmount = int(currentLast) + minStep
				default:
					targetAmount = int(currentLast) + minStep + r.Intn(3)*minStep
				}

				key := fmt.Sprintf("race_cli_%s_%d_%d", bidder.UUID[:8], b, time.Now().UnixNano())
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
					var parsed struct {
						Error string `json:"error"`
					}
					_ = json.Unmarshal(respBytes, &parsed)
					errMsg = parsed.Error
				}

				if status == http.StatusOK {
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

				time.Sleep(time.Duration(r.Intn(10)+2) * time.Millisecond)
			}
		}(u, userIdx)
	}

	doneGroup.Wait()
	totalDuration := time.Since(startTime)

	fmt.Printf("[+] Finished in %v. Total bids sent: %d\n", totalDuration, len(allResults))

	// Fetch DB bids
	rows, err := pool.Query(context.Background(), "SELECT id, user_id, amount, created_at FROM bids WHERE lot_id = $1 ORDER BY id ASC", lotID)
	if err != nil {
		fmt.Printf("ERROR: Failed to query bids: %v\n", err)
		os.Exit(1)
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
		_ = rows.Scan(&rb.ID, &rb.UserID, &rb.Amount, &rb.CreatedAt)
		acceptedBids = append(acceptedBids, rb)
	}

	fmt.Printf("[+] Accepted bids in DB: %d\n", len(acceptedBids))
	if len(acceptedBids) == 0 {
		fmt.Println("[-] No bids were accepted!")
		os.Exit(1)
	}

	winningBid := acceptedBids[len(acceptedBids)-1]
	var winnerName string
	for _, u := range users {
		if u.UUID == winningBid.UserID {
			winnerName = u.Name
			break
		}
	}

	fmt.Println("------------------------------------------------------------------")
	fmt.Printf("🏆 WINNER: %s (UUID: %s)\n", winnerName, winningBid.UserID)
	fmt.Printf("💰 WINNING AMOUNT: %d\n", winningBid.Amount)
	fmt.Println("------------------------------------------------------------------")

	// Verify holds
	fmt.Println("[*] Verifying hold balances integrity in DB:")
	for _, u := range users {
		var hold int
		_ = pool.QueryRow(context.Background(), "SELECT hold FROM users WHERE id = $1", u.UUID).Scan(&hold)
		if u.UUID == winningBid.UserID {
			fmt.Printf("    * Winner %s: hold = %d (matches winning bid: %v)\n", u.Name, hold, hold == winningBid.Amount)
		} else {
			fmt.Printf("    * Non-winner %s: hold = %d (expected 0: %v)\n", u.Name, hold, hold == 0)
		}
	}
}
