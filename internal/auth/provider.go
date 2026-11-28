package auth

import (
	"fmt"
	"strings"
)

const (
	// ServiceName is the OS keyring service used for gh-aipr secrets.
	// It must not collide with go-git-aico's "git-aico".
	ServiceName = "gh-aipr"

	ProviderOpenAI = "openai"

	appName = "gh-aipr"
)

// Providers is the fixed set of providers that can be registered.
var Providers = []string{ProviderOpenAI}

// NormalizeProvider lowercases and validates a provider name.
func NormalizeProvider(s string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(s))
	switch p {
	case ProviderOpenAI:
		return p, nil
	case "":
		return "", fmt.Errorf("provider is required (openai)")
	case "anthropic":
		return "", fmt.Errorf("anthropic is no longer supported; this tool is OpenAI-only")
	default:
		return "", fmt.Errorf("unknown provider %q (supported: openai)", s)
	}
}

// EnvKeyName returns the environment variable that overrides a provider's keyring secret.
func EnvKeyName(provider string) string {
	switch provider {
	case ProviderOpenAI:
		return "OPENAI_API_KEY"
	default:
		return ""
	}
}

// MissingKeyError explains how to supply an OpenAI API key.
func MissingKeyError() error {
	return fmt.Errorf("OPENAI_API_KEY is required. Set OPENAI_API_KEY, or run: gh aipr auth register")
}
