package main

import (
	"fmt"
	"strings"
)

// AskAI is a unified function that routes requests to either OpenAI or Anthropic API
// based on the configuration
func AskAI(config Config, question string, verbose bool) (string, error) {
	// Determine which provider to use
	provider := config.Provider
	if provider == "" {
		provider = "openai" // Default to OpenAI if not specified
	}
	
	provider = strings.ToLower(provider)
	
	// Route based on provider
	switch provider {
	case "openai":
		// Determine which model to use
		model := config.OpenAIModel
		
		// Check if API key is available
		if config.OpenAIKey == "" {
			return "", fmt.Errorf("OpenAI API key not found. Set the OPENAI_API_KEY environment variable")
		}
		
		return AskOpenAI(openAIURL, config.OpenAIKey, model, config.OpenAITemperature, config.OpenAIMaxTokens, question, verbose)
		
	case "anthropic":
		// Determine which model to use
		model := config.AnthropicModel
		
		// Check if API key is available
		if config.AnthropicKey == "" {
			return "", fmt.Errorf("Anthropic API key not found. Set the ANTHROPIC_API_KEY environment variable")
		}
		
		return AskAnthropic(config.AnthropicKey, model, config.AnthropicTemperature, config.AnthropicMaxTokens, question, verbose)
		
	default:
		return "", fmt.Errorf("unsupported AI provider: %s. Use 'openai' or 'anthropic'", provider)
	}
}