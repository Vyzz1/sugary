package main

import "testing"

func TestNormalizeAIProvider(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":            "gemini",
		"gemini":      "gemini",
		"huggingface": "huggingface",
		"huggie":      "huggingface",
	}

	for input, expected := range tests {
		if got := normalizeAIProvider(input); got != expected {
			t.Fatalf("normalizeAIProvider(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestAIFallbackEnabledRequiresDifferentExplicitProvider(t *testing.T) {
	t.Parallel()

	primaryProvider := normalizeAIProvider("gemini")
	fallbackProvider := normalizeAIProvider("huggingface")
	fallbackEnabled := true && fallbackProvider != "" && fallbackProvider != primaryProvider
	if !fallbackEnabled {
		t.Fatal("expected gemini primary with huggingface fallback to enable fallback wrapper")
	}

	fallbackProvider = normalizeOptionalAIProvider("")
	fallbackEnabled = true && fallbackProvider != "" && fallbackProvider != primaryProvider
	if fallbackEnabled {
		t.Fatal("expected empty fallback provider not to enable fallback wrapper")
	}

	fallbackProvider = normalizeAIProvider("gemini")
	fallbackEnabled = true && fallbackProvider != "" && fallbackProvider != primaryProvider
	if fallbackEnabled {
		t.Fatal("expected same primary/fallback provider not to enable fallback wrapper")
	}
}
