package caiyun

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"handy-translate/config"
)

func TestPostQueryContextReturnsHTTPStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	provider := &Caiyun{Translate: config.Translate{
		Key:     "test-key",
		BaseURL: server.URL,
	}}
	_, err := provider.PostQueryContext(context.Background(), "hello", "en", "zh")
	if err == nil {
		t.Fatal("expected non-200 response to return an error")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("error = %q, want status code", err)
	}
}

func TestPostQueryContextUsesCallerCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	provider := &Caiyun{Translate: config.Translate{
		Key:     "test-key",
		BaseURL: server.URL,
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := provider.PostQueryContext(ctx, "hello", "en", "zh")
	if err == nil {
		t.Fatal("expected canceled context to return an error")
	}
}
