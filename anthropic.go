package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const anthropicURL = "https://api.anthropic.com/v1/messages"

type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Content []struct {
		Type  string `json:"type"`
		Text  string `json:"text"`
	} `json:"content"`
}

func AskAnthropic(apiKey, model string, temperature float64, maxTokens int, question string, verbose bool) (string, error) {
	// Start the spinner
	spinner := NewSpinner("Asking Anthropic")
	spinner.Start()
	defer spinner.Stop()

	data := AnthropicRequest{
		Messages:    []AnthropicMessage{{Role: "user", Content: question}},
		Model:       model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", anthropicURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK HTTP status from Anthropic: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if verbose {
		fmt.Printf("\nRaw response from Anthropic: %v", string(respBody))
	}

	var apiResp AnthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Content) > 0 {
		var responseBuilder strings.Builder
		for _, content := range apiResp.Content {
			if content.Type == "text" {
				responseBuilder.WriteString(content.Text)
			}
		}
		return strings.TrimSpace(responseBuilder.String()), nil
	}

	return "", fmt.Errorf("no response from Anthropic")
}