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

const requestCount = 2000

func fetch(client *http.Client, id int) Result {
	start := time.Now()

	resp, err := client.Get("http://127.0.0.1:8080/work")
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
	queueSize := flag.Int("queuesize", 100, "jobs channel buffer size")
	rate := flag.Int("rate", 400, "incoming requests per second")
	flag.Parse()

	interval := time.Second / time.Duration(*rate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

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
			<-ticker.C
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

	throughput := float64(successful) / totalTime.Seconds()
	fmt.Printf("actual throughput: %.2f requests/sec\n", throughput)

	_, _, avg, _, p95, _ := getStats(queueTimes)
	fmt.Printf("QueueTime Avg/P95: %v / %v\n", avg, p95)

	_, _, avg, _, p95, _ = getStats(totalTimes)
	fmt.Printf("TotalTime Avg/P95: %v / %v\n", avg, p95)

	_, _, avg, _, p95, _ = getStats(serviceTimes)
	fmt.Printf("ServiceTime Avg/P95: %v / %v\n", avg, p95)
}
