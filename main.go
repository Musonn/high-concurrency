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
	QueueTime time.Duration
	TotalTime time.Duration
}

type Job struct {
	CreatedAt time.Time
}

type AdmissionStats struct {
	Accepted int
	Rejected int
	Duration time.Duration
}

const requestCount = 2000

func validateConfig(workerCount, queueSize, rate int) error {
	switch {
	case workerCount <= 0:
		return errors.New("workers must be greater than zero")
	case queueSize < 0:
		return errors.New("queuesize must not be negative")
	case rate <= 0:
		return errors.New("rate must be greater than zero")
	default:
		return nil
	}
}

func fetch(client *http.Client) {
	resp, err := client.Get("http://127.0.0.1:8080/work")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	_, _ = io.Copy(io.Discard, resp.Body)
}

func worker(client *http.Client, jobs <-chan Job, results chan<- Result) {
	for job := range jobs {
		queueTime := time.Since(job.CreatedAt)
		fetch(client)
		results <- Result{
			QueueTime: queueTime,
			TotalTime: time.Since(job.CreatedAt),
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

func main() {
	workerCount := flag.Int("workers", 10, "number of concurrent workers")
	queueSize := flag.Int("queuesize", 100, "jobs channel buffer size")
	rate := flag.Int("rate", 400, "incoming requests per second")
	flag.Parse()
	if err := validateConfig(*workerCount, *queueSize, *rate); err != nil {
		fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
		os.Exit(2)
	}

	interval := time.Second / time.Duration(*rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	jobs := make(chan Job, *queueSize)
	results := make(chan Result)
	admissionStats := make(chan AdmissionStats, 1)
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for i := 0; i < *workerCount; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			worker(client, jobs, results)
		}()
	}

	go func() {
		accepted := 0
		rejected := 0
		producerStart := time.Now()

		for i := 0; i < requestCount; i++ {
			<-ticker.C
			job := Job{
				CreatedAt: time.Now(),
			}

			select {
			case jobs <- job:
				accepted++
			default:
				rejected++
			}
		}

		producerDuration := time.Since(producerStart)

		close(jobs)

		admissionStats <- AdmissionStats{
			Accepted: accepted,
			Rejected: rejected,
			Duration: producerDuration,
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect latency samples for accepted jobs.
	var queueTimes []time.Duration
	var totalTimes []time.Duration

	for result := range results {
		queueTimes = append(queueTimes, result.QueueTime)
		totalTimes = append(totalTimes, result.TotalTime)
	}

	stats := <-admissionStats
	fmt.Printf("Offered rate: %d requests/sec\n", *rate)
	fmt.Printf("Actual offered rate: %.2f requests/sec\n", float64(requestCount)/stats.Duration.Seconds())
	fmt.Printf("Accepted: %d\n", stats.Accepted)
	fmt.Printf("Rejected: %d\n", stats.Rejected)
	fmt.Printf("Reject %%: %.2f%%\n", 100*float64(stats.Rejected)/float64(requestCount))
	fmt.Printf("Accepted rate: %.2f requests/sec\n", float64(stats.Accepted)/stats.Duration.Seconds())
	fmt.Printf("Queue P95: %v\n", p95(queueTimes))
	fmt.Printf("Total P95: %v\n", p95(totalTimes))
}
