package service

import (
	"context"
	"testing"

	"handy-translate/config"
	"handy-translate/internal/translate"
)

type staticProvider struct {
	result []string
}

func (p *staticProvider) Name() string {
	return "static"
}

func (p *staticProvider) Translate(context.Context, translate.TranslateRequest) ([]string, error) {
	return p.result, nil
}

type recordingEvents struct {
	result string
}

func (e *recordingEvents) EmitResult(result string)        { e.result = result }
func (e *recordingEvents) EmitResultStream(string)         {}
func (e *recordingEvents) EmitResultMeaningsStream(string) {}
func (e *recordingEvents) EmitStreamDone()                 {}
func (e *recordingEvents) EmitStreamError(string)          {}
func (e *recordingEvents) EmitWordQueryResult(string)      {}

func TestTranslateEmitsNormalProviderResult(t *testing.T) {
	registry := translate.NewRegistry()
	registry.Register("static", func(config.Translate) translate.Provider {
		return &staticProvider{result: []string{"你好", "世界"}}
	})

	cfg := &config.Config{
		TranslateWay: "static",
		Translate: map[string]config.Translate{
			"static": {Name: "Static", Key: "test-key"},
		},
	}
	events := &recordingEvents{}
	translator := NewTranslator(registry, cfg, events, nil, nil)

	result := translator.Translate(context.Background(), "hello", "en", "zh")
	if result != "你好\n世界" {
		t.Fatalf("Translate() result = %q", result)
	}
	if events.result != result {
		t.Fatalf("EmitResult() = %q, want %q", events.result, result)
	}
}
