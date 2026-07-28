package app

import (
	"strings"
	"testing"

	"handy-translate/config"
)

func TestGetTranslateMapJSONDoesNotExposeCredentials(t *testing.T) {
	original := config.Data
	config.Data = config.Config{
		Translate: map[string]config.Translate{
			"example": {
				Name:    "Example",
				AppID:   "private-app-id",
				Key:     "private-api-key",
				BaseURL: "https://private.example",
				Model:   "private-model",
			},
		},
	}
	t.Cleanup(func() {
		config.Data = original
	})

	result := GetTranslateMapJSON()
	for _, secret := range []string{
		"private-app-id",
		"private-api-key",
		"https://private.example",
		"private-model",
	} {
		if strings.Contains(result, secret) {
			t.Fatalf("public provider JSON exposed %q: %s", secret, result)
		}
	}
	if !strings.Contains(result, `"name":"Example"`) {
		t.Fatalf("public provider name missing: %s", result)
	}
}
