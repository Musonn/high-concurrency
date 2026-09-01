package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type Result struct {
	ID          int
	QueueTime   time.Duration
	ServiceTime time.Duration
	TotalTime   time.Duration
	Err         error
}

type Job struct {
	ID        int
	CreatedAt time.Time
}

const requestCount = 100

func fetch(client *http.Client, id int) Result {
	start := time.Now()

	resp, err := client.Get(fmt.Sprintf("https://jsonplaceholder.typicode.com/posts/%d", id))
	if err != nil {
		fmt.Printf("failed to fetch post %d: %v\n", id, err)
		return Result{ID: id, ServiceTime: time.Since(start), Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("post %d returned status %d\n", id, resp.StatusCode)
		return Result{ID: id, ServiceTime: time.Since(start), Err: fmt.Errorf("status code %d", resp.StatusCode)}
	}

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return Result{
			ID:          id,
			ServiceTime: time.Since(start),
			Err:         err,
		}
	}

	return Result{ID: id, ServiceTime: time.Since(start)}
}

func worker(client *http.Client, jobs <-chan Job, results chan<- Result) {
	for job := range jobs {
		queueTime := time.Since(job.CreatedAt)
		result := fetch(client, job.ID)
		result.QueueTime = queueTime
		result.TotalTime = time.Since(job.CreatedAt)
		results <- result
	}
}

func getStats(durations []time.Duration) (min, max, avg, p50, p95, p99 time.Duration) {
	if len(durations) == 0 {
		return
	}

	var total time.Duration

	for _, d := range durations {
		total += d
	}

	avg = total / time.Duration(len(durations))
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})
	min = durations[0]
	max = durations[len(durations)-1]
	p50 = percentile(durations, 50)
	p95 = percentile(durations, 95)
	p99 = percentile(durations, 99)

	return min, max, avg, p50, p95, p99
}

func percentile(sortedDurations []time.Duration, percentile int) time.Duration {
	index := (len(sortedDurations)*percentile + 99) / 100
	if index > 0 {
		index--
	}
	return sortedDurations[index]
}

func main() {
	workerCount := flag.Int("workers", 10, "number of concurrent workers")
	queueSize := flag.Int("queuesize", 10, "jobs channel buffer size")
	flag.Parse()

	jobs := make(chan Job, *queueSize)
	results := make(chan Result)
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

	start := time.Now()

	go func() {
		for id := 1; id <= requestCount; id++ {
			jobs <- Job{ID: id, CreatedAt: time.Now()}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and calculate statistics.
	successful, failed := 0, 0
	var queueTimes []time.Duration
	var serviceTimes []time.Duration
	var totalTimes []time.Duration

	for result := range results {
		queueTimes = append(queueTimes, result.QueueTime)
		serviceTimes = append(serviceTimes, result.ServiceTime)
		totalTimes = append(totalTimes, result.TotalTime)

		if result.Err != nil {
			failed++
		} else {
			successful++
		}
	}
	totalTime := time.Since(start)

	fmt.Printf("Successful: %d, Failed: %d\n", successful, failed)

	throughput := float64(successful) / totalTime.Seconds()
	fmt.Printf("Total time taken: %v\n", totalTime)
	fmt.Printf("Throughput: %.2f requests/sec\n", throughput)

	min, max, avg, p50, p95, p99 := getStats(queueTimes)
	fmt.Printf("Queue Time - Min: %v, Max: %v, Avg: %v, P50: %v, P95: %v, P99: %v\n", min, max, avg, p50, p95, p99)

	min, max, avg, p50, p95, p99 = getStats(serviceTimes)
	fmt.Printf("Service Time - Min: %v, Max: %v, Avg: %v, P50: %v, P95: %v, P99: %v\n", min, max, avg, p50, p95, p99)

	min, max, avg, p50, p95, p99 = getStats(totalTimes)
	fmt.Printf("Total Time - Min: %v, Max: %v, Avg: %v, P50: %v, P95: %v, P99: %v\n", min, max, avg, p50, p95, p99)
}
