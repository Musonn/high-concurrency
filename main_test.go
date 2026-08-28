package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testClient(handler roundTripFunc) *http.Client {
	return &http.Client{Transport: handler}
}

func response(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchPost(t *testing.T) {
	client := testClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/posts/42" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/posts/42")
		}

		return response(http.StatusOK, `{"userId":7,"id":42,"title":"hello","body":"world"}`), nil
	})

	post, err := fetchPost(context.Background(), client, "https://example.test", 42)
	if err != nil {
		t.Fatalf("fetchPost() error = %v", err)
	}

	if post.ID != 42 || post.UserID != 7 || post.Title != "hello" || post.Body != "world" {
		t.Fatalf("fetchPost() = %+v, want the test post", post)
	}
}

func TestFetchPostRejectsUnexpectedStatus(t *testing.T) {
	client := testClient(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusServiceUnavailable, "unavailable"), nil
	})

	_, err := fetchPost(context.Background(), client, "https://example.test", 1)
	if err == nil || !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Fatalf("fetchPost() error = %v, want an HTTP 503 error", err)
	}
}

func TestFetchPostRejectsInvalidJSON(t *testing.T) {
	client := testClient(func(_ *http.Request) (*http.Response, error) {
		return response(http.StatusOK, "not JSON"), nil
	})

	_, err := fetchPost(context.Background(), client, "https://example.test", 1)
	if err == nil || !strings.Contains(err.Error(), "decode post 1") {
		t.Fatalf("fetchPost() error = %v, want a decode error", err)
	}
}
