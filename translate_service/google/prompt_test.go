package google

import (
	"strings"
	"testing"
)

func TestBuildTranslateMessagesIncludesLanguageDirection(t *testing.T) {
	messages := buildTranslateMessages(TranslatePrompts, "bonjour", "fr", "ja")
	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(messages))
	}
	for _, expected := range []string{"fr", "ja"} {
		if !strings.Contains(messages[0].Content, expected) {
			t.Fatalf("system prompt %q does not contain %q", messages[0].Content, expected)
		}
	}
}
