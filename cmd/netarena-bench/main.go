package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type simulationResponse struct {
	Name    string `json:"name"`
	Summary struct {
		TotalRequests     int `json:"totalRequests"`
		SucceededRequests int `json:"succeededRequests"`
		FailedRequests    int `json:"failedRequests"`
	} `json:"summary"`
}

type requestResult struct {
	latency time.Duration
	httpOK  bool
	valid   bool
	err     error
}

type batchSummary struct {
	throughput     float64
	p50            float64
	p95            float64
	httpErrors     int
	invalidResults int
}

type requestPayload struct {
	Name            string           `json:"name"`
	ServiceCapacity int              `json:"serviceCapacity"`
	ServiceDuration int              `json:"serviceDuration"`
	Arrivals        []requestArrival `json:"arrivals"`
}

type requestArrival struct {
	RequestID string `json:"requestId"`
	At        int    `json:"at"`
}

func main() {
	baseURL := flag.String("url", "http://127.0.0.1:18080", "NetArena API base URL")
	requests := flag.Int("requests", 20, "Measured requests per concurrency level")
	arrivals := flag.Int("arrivals", 20, "Arrivals in each simulation")
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout per HTTP request")
	runs := flag.Int("runs", 3, "Measured repetitions per concurrency level")
	flag.Parse()

	if *requests < 1 || *arrivals < 1 || *runs < 1 || *timeout <= 0 {
		fmt.Println("requests, arrivals, and runs must be positive, and timeout must be greater than zero")
		os.Exit(1)
	}

	client := &http.Client{Timeout: *timeout}
	ctx := context.Background()

	if err := checkHealth(ctx, client, *baseURL); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("NetArena API benchmark: %s\n", *baseURL)
	fmt.Printf(
		"Measured requests per level: %d; arrivals per simulation: %d; repetitions: %d\n",
		*requests,
		*arrivals,
		*runs,
	)
	fmt.Println("Concurrency | Throughput req/s | p50 ms | p95 ms | HTTP errors | Invalid results")

	for _, concurrency := range []int{1, 4, 8} {
		warmup := sendRequest(
			ctx, client, *baseURL, *arrivals,
			fmt.Sprintf("warmup-c%d", concurrency),
		)
		if !warmup.httpOK || !warmup.valid {
			fmt.Printf("Warm-up failed at concurrency %d: %v\n", concurrency, warmup.err)
			os.Exit(1)
		}

		throughputs := make([]float64, 0, *runs)
		p50s := make([]float64, 0, *runs)
		p95s := make([]float64, 0, *runs)
		totalHTTPErrors := 0
		totalInvalidResults := 0

		for run := 1; run <= *runs; run++ {
			fmt.Printf("Concurrency %d — run %d/%d\n", concurrency, run, *runs)

			start := time.Now()
			results := runBatch(
				ctx, client, *baseURL, concurrency, *requests, *arrivals,
			)
			elapsed := time.Since(start)

			summary := summarizeBatch(results, elapsed)
			for i, result := range results {
				if result.err != nil {
					fmt.Printf(
						"  Request %d failed (HTTP OK: %t, valid: %t): %v\n",
						i+1,
						result.httpOK,
						result.valid,
						result.err,
					)
				} else if !result.valid {
					fmt.Printf("  Request %d returned an invalid simulation response\n", i+1)
				}
			}
			throughputs = append(throughputs, summary.throughput)
			p50s = append(p50s, summary.p50)
			p95s = append(p95s, summary.p95)
			totalHTTPErrors += summary.httpErrors
			totalInvalidResults += summary.invalidResults
		}

		fmt.Printf(
			"%11d | %16.2f | %6.2f | %6.2f | %11d | %14d\n",
			concurrency,
			percentile(throughputs, 0.50),
			percentile(p50s, 0.50),
			percentile(p95s, 0.50),
			totalHTTPErrors,
			totalInvalidResults,
		)
	}
}

func checkHealth(ctx context.Context, client *http.Client, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/health", nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func sendRequest(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	arrivalCount int,
	name string,
) requestResult {
	arrivals := make([]requestArrival, arrivalCount)
	for i := range arrivals {
		arrivals[i] = requestArrival{
			RequestID: fmt.Sprintf("%s-req-%d", name, i),
			At:        i,
		}
	}

	payload := requestPayload{
		Name:            name,
		ServiceCapacity: 2,
		ServiceDuration: 3,
		Arrivals:        arrivals,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return requestResult{err: err}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		baseURL+"/api/v1/simulations",
		bytes.NewReader(data),
	)
	if err != nil {
		return requestResult{err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return requestResult{latency: time.Since(start), err: err}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	latency := time.Since(start)
	if err != nil {
		return requestResult{
			latency: latency,
			httpOK:  true,
			valid:   false,
			err:     err,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return requestResult{
			latency: latency,
			httpOK:  false,
			valid:   false,
			err:     fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)),
		}
	}

	var body simulationResponse
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return requestResult{
			latency: latency,
			httpOK:  true,
			valid:   false,
			err:     err,
		}
	}

	valid := validateResponse(body, name, arrivalCount)

	return requestResult{
		latency: latency,
		httpOK:  true,
		valid:   valid,
	}
}

func runBatch(
	ctx context.Context,
	client *http.Client,
	baseURL string,
	concurrency int,
	requestCount int,
	arrivalCount int,
) []requestResult {
	results := make([]requestResult, requestCount)
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			name := fmt.Sprintf("bench-c%d-r%d", concurrency, index)
			results[index] = sendRequest(ctx, client, baseURL, arrivalCount, name)
		}(i)
	}

	wg.Wait()

	return results
}

func summarizeBatch(results []requestResult, elapsed time.Duration) batchSummary {
	latencies := make([]float64, 0, len(results))
	summary := batchSummary{}

	for _, result := range results {
		if !result.httpOK {
			summary.httpErrors++
			continue
		}

		if !result.valid {
			summary.invalidResults++
		}

		latencies = append(
			latencies,
			float64(result.latency)/float64(time.Millisecond),
		)
	}

	if elapsed > 0 {
		summary.throughput = float64(len(results)) / elapsed.Seconds()
	}

	summary.p50 = percentile(latencies, 0.50)
	summary.p95 = percentile(latencies, 0.95)

	return summary
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	index := int(float64(len(values))*p+0.999999) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func validateResponse(body simulationResponse, name string, arrivalCount int) bool {
	return body.Name == name &&
		body.Summary.TotalRequests == arrivalCount &&
		body.Summary.SucceededRequests == arrivalCount &&
		body.Summary.FailedRequests == 0
}
