package main

import "fmt"

// chatCompleter returns the assistant message content for the given messages.
type chatCompleter func(messages []OpenAIMessage) (string, error)

func generatePRFromConfig(cfg Config, diff string, japanise, verbose bool) (string, string, error) {
	complete := func(messages []OpenAIMessage) (string, error) {
		return AskAI(cfg, messages, verbose)
	}
	return generatePR(complete, diff, japanise)
}

func generatePR(complete chatCompleter, diff string, japanise bool) (string, string, error) {
	messages := []OpenAIMessage{
		{Role: "system", Content: systemPrompt(japanise)},
		{Role: "user", Content: diff},
	}
	raw, err := complete(messages)
	if err != nil {
		return "", "", err
	}
	title, body, verr := parseAssembleSanitizeValidate(raw)
	if verr == nil {
		return title, body, nil
	}

	retryMessages := []OpenAIMessage{
		{Role: "system", Content: systemPrompt(japanise)},
		{Role: "user", Content: diff},
		{Role: "assistant", Content: raw},
		{Role: "user", Content: retryFixInstruction(verr, japanise)},
	}
	raw, err = complete(retryMessages)
	if err != nil {
		return "", "", err
	}
	title, body, verr = parseAssembleSanitizeValidate(raw)
	if verr != nil {
		return "", "", fmt.Errorf("AI output failed validation after retry: %w", verr)
	}
	return title, body, nil
}
