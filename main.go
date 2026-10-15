package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type Result struct {
	ID          int
	StatusCode  int
	ServiceTime time.Duration
	Err         error
}

type Job struct {
	ID        int
	CreatedAt time.Time
}

const requestCount = 2000
const maxRetries = 3

var offeredRates = []int{300, 400, 500, 600, 800}

type attemptResult struct {
	Result
	Final bool
}

func validateConfig(workerCount int) error {
	if workerCount <= 0 {
		return errors.New("workers must be greater than zero")
	}
	return nil
}

func fetch(client *http.Client) (int, error) {
	resp, err := client.Get("http://127.0.0.1:8080/work")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}

func worker(client *http.Client, jobs <-chan Job, results chan<- attemptResult) {
	for job := range jobs {
		for retry := 0; ; retry++ {
			statusCode, err := fetch(client)
			shouldRetry := err == nil && statusCode == http.StatusServiceUnavailable && retry < maxRetries
			results <- attemptResult{
				Result: Result{
					ID:          job.ID,
					StatusCode:  statusCode,
					ServiceTime: time.Since(job.CreatedAt),
					Err:         err,
				},
				Final: !shouldRetry,
			}

			if !shouldRetry {
				break
			}
		}
	}
}

func p95(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	index := (len(durations)*95+99)/100 - 1
	return durations[index]
}

func runExperiment(workerCount, rate int) {
	interval := time.Second / time.Duration(rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Buffer every original request so local admission control cannot drop work.
	// This experiment measures the mock server's overload responses instead.
	jobs := make(chan Job, requestCount)
	results := make(chan attemptResult)
	productionDuration := make(chan time.Duration, 1)
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			worker(client, jobs, results)
		}()
	}

	go func() {
		producerStart := time.Now()

		for i := 0; i < requestCount; i++ {
			<-ticker.C
			job := Job{
				ID:        i,
				CreatedAt: time.Now(),
			}
			jobs <- job
		}

		close(jobs)
		productionDuration <- time.Since(producerStart)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var successfulLatencies []time.Duration
	var totalAttempts, okCount, unavailableCount, finalSuccesses, finalFailures int

	for attempt := range results {
		totalAttempts++

		switch {
		case attempt.Err == nil && attempt.StatusCode == http.StatusOK:
			okCount++
		case attempt.Err == nil && attempt.StatusCode == http.StatusServiceUnavailable:
			unavailableCount++
		}

		if !attempt.Final {
			continue
		}

		if attempt.Err == nil && attempt.StatusCode == http.StatusOK {
			finalSuccesses++
			successfulLatencies = append(successfulLatencies, attempt.ServiceTime)
		} else {
			finalFailures++
		}
	}

	duration := <-productionDuration
	retries := totalAttempts - requestCount
	fmt.Printf("Offered rate: %d requests/sec\n", rate)
	fmt.Printf("Actual offered rate: %.2f requests/sec\n", float64(requestCount)/duration.Seconds())
	fmt.Printf("Original requests: %d\n", requestCount)
	fmt.Printf("HTTP attempts: %d\n", totalAttempts)
	fmt.Printf("Retries: %d\n", retries)
	fmt.Printf("Amplification: %.2fx\n", float64(totalAttempts)/float64(requestCount))
	fmt.Printf("200 responses: %d\n", okCount)
	fmt.Printf("503 responses: %d\n", unavailableCount)
	fmt.Printf("Final success: %d\n", finalSuccesses)
	fmt.Printf("Final failure: %d\n", finalFailures)
	fmt.Printf("Success %%: %.2f%%\n", 100*float64(finalSuccesses)/float64(requestCount))
	fmt.Printf("Success P95: %v\n", p95(successfulLatencies))
}

func main() {
	workerCount := flag.Int("workers", 100, "number of concurrent workers")
	flag.Parse()
	if err := validateConfig(*workerCount); err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(2)
	}

	for _, rate := range offeredRates {
		runExperiment(*workerCount, rate)
		fmt.Println()
	}
}
