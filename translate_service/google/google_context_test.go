package google

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"handy-translate/config"
)

func TestPostQueryStreamContextCancelsRequest(t *testing.T) {
	started := make(chan struct{})
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-releaseHandler
	}))
	defer server.Close()
	defer close(releaseHandler)

	provider := &Google{Translate: config.Translate{
		Key:     "test-key",
		BaseURL: server.URL,
		Model:   "test-model",
	}}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- provider.PostQueryStreamContext(ctx, "hello", "en", "zh", func(string) {})
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}
	cancel()

	select {
	case err := <-done:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not stop after cancellation")
	}
}
