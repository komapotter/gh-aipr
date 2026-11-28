package main

import "fmt"

// AskAI sends a prompt to the OpenAI Chat Completions API.
func AskAI(config Config, question string, verbose bool) (string, error) {
	if config.OpenAIKey == "" {
		return "", fmt.Errorf("OpenAI API key not found. Set OPENAI_API_KEY, or run: gh aipr auth register")
	}

	return AskOpenAI(openAIURL, config.OpenAIKey, config.OpenAIModel, config.OpenAITemperature, config.OpenAIMaxTokens, config.OpenAIReasoningEffort, question, verbose)
}
