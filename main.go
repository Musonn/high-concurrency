package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func fetch(id int) {
	resp, err := http.Get(fmt.Sprintf("https://jsonplaceholder.typicode.com/posts/%d", id))
	if err != nil {
		fmt.Printf("failed to fetch post %d: %v\n", id, err)
		return
	}
	defer resp.Body.Close()

	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		fmt.Printf("failed to decode post %d: %v\n", id, err)
		return
	}

	fmt.Printf("%d: %s\n", post.ID, post.Title)
}

func worker(jobs <-chan int) {
	for id := range jobs {
		fetch(id)
	}
}

func main() {
	jobs := make(chan int)
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			worker(jobs)
		}()
	}

	for id := 1; id <= 100; id++ {
		jobs <- id
	}

	close(jobs)
	wg.Wait()
}
