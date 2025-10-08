package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

const baseURL = "http://localhost:8080/api/v1"

// ==============================================
// REQUEST MODELS (Match your API exactly)
// ==============================================

type DepositRequest struct {
	UserID         int    `json:"user_id"`
	Amount         int64  `json:"amount"` // Amount in KOBO, not Naira!
	IdempotencyKey string `json:"idempotency_key"`
	Reference      string `json:"reference,omitempty"`
}

type WithdrawRequest struct {
	UserID         int    `json:"user_id"`
	Amount         int64  `json:"amount"` // Amount in KOBO
	IdempotencyKey string `json:"idempotency_key"`
	Reference      string `json:"reference,omitempty"`
}

type TransferRequest struct {
	FromUserID     int    `json:"from_user_id"` // Fixed field name!
	ToUserID       int    `json:"to_user_id"`   // Fixed field name!
	Amount         int64  `json:"amount"`       // Amount in KOBO
	Fee            int64  `json:"fee,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
	Reference      string `json:"reference,omitempty"`
}

// ==============================================
// METRICS
// ==============================================

type Metrics struct {
	totalRequests   int64
	successRequests int64
	failedRequests  int64
	status400       int64
	status404       int64
	status422       int64
	status500       int64
	totalDuration   int64 // in milliseconds
}

var metrics Metrics

// ==============================================
// HELPER FUNCTIONS
// ==============================================

func checkHealth(client *http.Client) bool {
	// Try to get balance for user 1 as health check
	resp, err := client.Get(baseURL + "/balance/1")
	if err != nil {
		fmt.Println("❌ Health check failed:", err)
		return false
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ Health check passed: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Response: %s\n", string(body))
	}
	return resp.StatusCode == http.StatusOK
}

func sendRequest(client *http.Client, method, url string, body interface{}, requestType string) {
	atomic.AddInt64(&metrics.totalRequests, 1)

	data, _ := json.Marshal(body)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		atomic.AddInt64(&metrics.failedRequests, 1)
		fmt.Printf("❌ Request creation error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start).Milliseconds()
	atomic.AddInt64(&metrics.totalDuration, duration)

	if err != nil {
		atomic.AddInt64(&metrics.failedRequests, 1)
		fmt.Printf("❌ Connection error [%s]: %v\n", requestType, err)
		return
	}
	defer resp.Body.Close()

	// Read response body for debugging
	responseBody, _ := io.ReadAll(resp.Body)

	// Track status codes
	switch resp.StatusCode {
	case http.StatusOK:
		atomic.AddInt64(&metrics.successRequests, 1)
		fmt.Printf("✅ %s %s -> %d (%dms)\n", method, requestType, resp.StatusCode, duration)
	case http.StatusBadRequest:
		atomic.AddInt64(&metrics.status400, 1)
		atomic.AddInt64(&metrics.failedRequests, 1)
		fmt.Printf("⚠️  %s %s -> 400 BAD REQUEST (%dms)\n   Body: %s\n", method, requestType, duration, string(responseBody))
	case http.StatusNotFound:
		atomic.AddInt64(&metrics.status404, 1)
		atomic.AddInt64(&metrics.failedRequests, 1)
		fmt.Printf("⚠️  %s %s -> 404 NOT FOUND (%dms)\n", method, requestType, duration)
	case http.StatusUnprocessableEntity:
		atomic.AddInt64(&metrics.status422, 1)
		atomic.AddInt64(&metrics.failedRequests, 1)
		fmt.Printf("⚠️  %s %s -> 422 UNPROCESSABLE (%dms): %s\n", method, requestType, duration, string(responseBody))
	case http.StatusInternalServerError:
		atomic.AddInt64(&metrics.status500, 1)
		atomic.AddInt64(&metrics.failedRequests, 1)
		fmt.Printf("❌ %s %s -> 500 SERVER ERROR (%dms): %s\n", method, requestType, duration, string(responseBody))
	default:
		atomic.AddInt64(&metrics.failedRequests, 1)
		fmt.Printf("⚠️  %s %s -> %d (%dms): %s\n", method, requestType, resp.StatusCode, duration, string(responseBody))
	}
}

func printMetrics() {
	total := atomic.LoadInt64(&metrics.totalRequests)
	success := atomic.LoadInt64(&metrics.successRequests)
	failed := atomic.LoadInt64(&metrics.failedRequests)
	totalDuration := atomic.LoadInt64(&metrics.totalDuration)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 LOAD TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Requests:     %d\n", total)
	fmt.Printf("Successful:         %d (%.1f%%)\n", success, float64(success)/float64(total)*100)
	fmt.Printf("Failed:             %d (%.1f%%)\n", failed, float64(failed)/float64(total)*100)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("400 Bad Request:    %d\n", atomic.LoadInt64(&metrics.status400))
	fmt.Printf("404 Not Found:      %d\n", atomic.LoadInt64(&metrics.status404))
	fmt.Printf("422 Unprocessable:  %d\n", atomic.LoadInt64(&metrics.status422))
	fmt.Printf("500 Server Error:   %d\n", atomic.LoadInt64(&metrics.status500))
	fmt.Println(strings.Repeat("-", 60))
	if total > 0 {
		fmt.Printf("Avg Response Time:  %dms\n", totalDuration/total)
	}
	fmt.Println(strings.Repeat("=", 60))
}

// ==============================================
// MAIN LOAD TEST
// ==============================================

func main() {
	// Configuration
	const concurrency = 10   // Number of concurrent goroutines
	const iterations = 50    // Requests per goroutine
	const startUserID = 1    // Start with user 1 (must exist in DB)

	fmt.Println("🚀 Starting Wallet API Concurrency Load Test")
	fmt.Printf("Configuration: %d goroutines × %d iterations = %d total requests\n\n", 
		concurrency, iterations, concurrency*iterations)

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Health check before starting
	fmt.Println("🔍 Running health check...")
	if !checkHealth(client) {
		fmt.Println("❌ Server is not healthy. Aborting load test.")
		os.Exit(1)
	}
	fmt.Println()

	// Give user time to read
	fmt.Println("Starting in 3 seconds...")
	time.Sleep(3 * time.Second)

	startTime := time.Now()
	var wg sync.WaitGroup

	// Launch concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			// Each worker uses a specific user ID
			userID := startUserID + workerID

			for j := 0; j < iterations; j++ {
				// Generate unique idempotency key for each request
				idempotencyKey := fmt.Sprintf("loadtest_%d_%d_%s", workerID, j, uuid.New().String())

				switch j % 3 {
				case 0:
					// Deposit ₦100 (10000 kobo)
					req := DepositRequest{
						UserID:         userID,
						Amount:        50000, // ₦100 in kobo
						IdempotencyKey: idempotencyKey,
						Reference:      fmt.Sprintf("load_test_deposit_%d_%d", workerID, j),
					}
					sendRequest(client, "POST", baseURL+"/deposit", req, fmt.Sprintf("DEPOSIT[Worker%d]", workerID))

				case 1:
					// Withdraw ₦50 (5000 kobo)
					req := WithdrawRequest{
						UserID:         userID,
						Amount:         10000, // ₦50 in kobo
						IdempotencyKey: idempotencyKey,
						Reference:      fmt.Sprintf("load_test_withdraw_%d_%d", workerID, j),
					}
					sendRequest(client, "POST", baseURL+"/withdraw", req, fmt.Sprintf("WITHDRAW[Worker%d]", workerID))

				case 2:
					// Transfer ₦25 (2500 kobo) to next user
					toUserID := startUserID + ((workerID + 1) % concurrency)
					req := TransferRequest{
						FromUserID:     userID,
						ToUserID:       toUserID,
						Amount:         11000, // ₦110 in kobo
						Fee:            500,  // ₦5 fee
						IdempotencyKey: idempotencyKey,
						Reference:      fmt.Sprintf("load_test_transfer_%d_%d", workerID, j),
					}
					sendRequest(client, "POST", baseURL+"/transfer", req, fmt.Sprintf("TRANSFER[Worker%d→%d]", workerID, toUserID-startUserID))
				}

				// Small delay to avoid overwhelming the server
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	totalTime := time.Since(startTime)

	// Print results
	fmt.Printf("\n⏱️  Total execution time: %v\n", totalTime)
	printMetrics()

	// Calculate throughput
	totalReqs := atomic.LoadInt64(&metrics.totalRequests)
	fmt.Printf("\n🚀 Throughput: %.2f requests/second\n", float64(totalReqs)/totalTime.Seconds())
}