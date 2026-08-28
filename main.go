package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const apiBaseURL = "https://jsonplaceholder.typicode.com"

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func fetchPost(ctx context.Context, client *http.Client, baseURL string, id int) (Post, error) {
	url := fmt.Sprintf("%s/posts/%d", strings.TrimRight(baseURL, "/"), id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Post{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Post{}, fmt.Errorf("fetch post %d: %w", id, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Post{}, fmt.Errorf("fetch post %d: unexpected HTTP status %s", id, resp.Status)
	}

	var post Post
	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		return Post{}, fmt.Errorf("decode post %d: %w", id, err)
	}

	return post, nil
}

func run() error {
	client := &http.Client{Timeout: 10 * time.Second}
	post, err := fetchPost(context.Background(), client, apiBaseURL, 1)
	if err != nil {
		return err
	}

	fmt.Printf("Post #%d by user %d\nTitle: %s\n\n%s\n", post.ID, post.UserID, post.Title, post.Body)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
