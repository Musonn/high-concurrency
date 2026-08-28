package main

import (
	"encoding/json"
	"fmt"
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

func fetch(client *http.Client, id int) {
	resp, err := client.Get(fmt.Sprintf("https://jsonplaceholder.typicode.com/posts/%d", id))
	if err != nil {
		fmt.Printf("failed to fetch post %d: %v\n", id, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("post %d returned status %d\n", id, resp.StatusCode)
		return
	}

	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		fmt.Printf("failed to decode post %d: %v\n", id, err)
		return
	}

	fmt.Printf("%d: %s\n", post.ID, post.Title)
}

func worker(client *http.Client, jobs <-chan int) {
	for id := range jobs {
		fetch(client, id)
	}
}

func main() {
	jobs := make(chan int)
	var wg sync.WaitGroup

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			worker(client, jobs)
		}()
	}

	start := time.Now()

	for id := 1; id <= 100; id++ {
		jobs <- id
	}

	close(jobs)
	wg.Wait()

	fmt.Printf("Total time: %s\n", time.Since(start))
}
