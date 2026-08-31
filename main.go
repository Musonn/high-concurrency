package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
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
	ID       int
	Duration time.Duration
	Err      error
}

func fetch(client *http.Client, id int) Result {
	start := time.Now()

	resp, err := client.Get(fmt.Sprintf("https://jsonplaceholder.typicode.com/posts/%d", id))
	if err != nil {
		fmt.Printf("failed to fetch post %d: %v\n", id, err)
		return Result{ID: id, Duration: time.Since(start), Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("post %d returned status %d\n", id, resp.StatusCode)
		return Result{ID: id, Duration: time.Since(start), Err: fmt.Errorf("status code %d", resp.StatusCode)}
	}

	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		fmt.Printf("failed to decode post %d: %v\n", id, err)
		return Result{ID: id, Duration: time.Since(start), Err: err}
	}

	return Result{ID: id, Duration: time.Since(start)}
}

func worker(client *http.Client, jobs <-chan int, results chan<- Result) {
	for id := range jobs {
		result := fetch(client, id)
		results <- result
	}
}

func main() {
	workerCount := flag.Int("workers", 10, "number of concurrent workers")
	flag.Parse()
	if *workerCount <= 0 {
		log.Fatal("workers must be greater than 0")
	}

	jobs := make(chan int)
	results := make(chan Result)
	var wg sync.WaitGroup
	start := time.Now()

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
		for id := 1; id <= 100; id++ {
			jobs <- id
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	successful, failed := 0, 0
	var durations []time.Duration

	for result := range results {
		durations = append(durations, result.Duration)
		if result.Err != nil {
			failed++
		} else {
			successful++
		}
	}

	fmt.Printf("Successful: %d, Failed: %d\n", successful, failed)
	fmt.Printf("Durations: ")
	for _, duration := range durations {
		fmt.Printf("%s ", duration)
	}
	fmt.Println()
	fmt.Printf("Total time: %s\n", time.Since(start))
}
