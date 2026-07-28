package deepseek

import (
	"strings"
	"testing"
)

func TestFormatTranslationPromptIncludesLanguageDirection(t *testing.T) {
	prompt, err := formatTranslationPrompt("bonjour", "fr", "ja")
	if err != nil {
		t.Fatalf("formatTranslationPrompt() error = %v", err)
	}
	for _, expected := range []string{"fr", "ja", "bonjour"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt %q does not contain %q", prompt, expected)
		}
	}
}
