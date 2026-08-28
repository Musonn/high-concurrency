package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func main() {
	resp, err := http.Get(
		"https://jsonplaceholder.typicode.com/posts/1",
	)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var post Post

	if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
		panic(err)
	}

	fmt.Printf(
	"ID: %d\nUser ID: %d\nTitle: %s\nBody: %s\n",
	post.ID,
	post.UserID,
	post.Title,
	post.Body,
)
}
