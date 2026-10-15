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

var offeredRates = []int{300, 400, 500, 600, 800}

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

func worker(client *http.Client, jobs <-chan Job, results chan<- Result) {
	for job := range jobs {
		statusCode, err := fetch(client)
		results <- Result{
			ID:          job.ID,
			StatusCode:  statusCode,
			ServiceTime: time.Since(job.CreatedAt),
			Err:         err,
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
	results := make(chan Result)
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

	// Each job has one attempt in this baseline. Retrying will add additional
	// results with the same ID in the next experiment.
	var finalLatencies []time.Duration
	var totalAttempts, okCount, unavailableCount int

	for result := range results {
		totalAttempts++
		finalLatencies = append(finalLatencies, result.ServiceTime)

		switch {
		case result.Err == nil && result.StatusCode == http.StatusOK:
			okCount++
		case result.Err == nil && result.StatusCode == http.StatusServiceUnavailable:
			unavailableCount++
		}
	}

	duration := <-productionDuration
	finalFailures := requestCount - okCount
	retries := totalAttempts - requestCount
	fmt.Printf("Offered rate: %d requests/sec\n", rate)
	fmt.Printf("Actual offered rate: %.2f requests/sec\n", float64(requestCount)/duration.Seconds())
	fmt.Printf("Original requests: %d\n", requestCount)
	fmt.Printf("Total HTTP attempts: %d\n", totalAttempts)
	fmt.Printf("200 responses: %d\n", okCount)
	fmt.Printf("503 responses: %d\n", unavailableCount)
	fmt.Printf("Final failures: %d\n", finalFailures)
	fmt.Printf("Retries: %d\n", retries)
	fmt.Printf("P95 final latency: %v\n", p95(finalLatencies))
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
