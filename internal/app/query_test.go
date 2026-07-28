package app

import (
	"context"
	"testing"
	"time"

	"handy-translate/config"
	"handy-translate/internal/service"
	internaltranslate "handy-translate/internal/translate"
)

type cancelAwareProvider struct {
	firstStarted  chan struct{}
	firstCanceled chan struct{}
}

func (p *cancelAwareProvider) Name() string { return "cancel-aware" }

func (p *cancelAwareProvider) Translate(ctx context.Context, req internaltranslate.TranslateRequest) ([]string, error) {
	if req.Text == "first" {
		close(p.firstStarted)
		<-ctx.Done()
		close(p.firstCanceled)
		return nil, ctx.Err()
	}
	return []string{"second-result"}, nil
}

type queryEvents struct {
	result chan string
}

func (e *queryEvents) EmitResult(result string)        { e.result <- result }
func (e *queryEvents) EmitResultStream(string)         {}
func (e *queryEvents) EmitResultMeaningsStream(string) {}
func (e *queryEvents) EmitStreamDone()                 {}
func (e *queryEvents) EmitStreamError(string)          {}
func (e *queryEvents) EmitWordQueryResult(string)      {}

func TestStartCurrentQueryCancelsPreviousQuery(t *testing.T) {
	provider := &cancelAwareProvider{
		firstStarted:  make(chan struct{}),
		firstCanceled: make(chan struct{}),
	}
	registry := internaltranslate.NewRegistry()
	registry.Register("cancel-aware", func(config.Translate) internaltranslate.Provider {
		return provider
	})
	cfg := &config.Config{
		TranslateWay: "cancel-aware",
		Translate: map[string]config.Translate{
			"cancel-aware": {Key: "test-key"},
		},
	}
	events := &queryEvents{result: make(chan string, 1)}
	translator := service.NewTranslator(registry, cfg, events, nil, nil)
	application := &Application{
		Translator: translator,
		State:      NewState(),
	}

	application.startCurrentQuery("first", "translate")
	select {
	case <-provider.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first query did not start")
	}

	application.startCurrentQuery("second", "translate")

	select {
	case <-provider.firstCanceled:
	case <-time.After(time.Second):
		t.Fatal("first query was not canceled")
	}
	select {
	case result := <-events.result:
		if result != "second-result" {
			t.Fatalf("result = %q, want %q", result, "second-result")
		}
	case <-time.After(time.Second):
		t.Fatal("second query did not complete")
	}
}
