package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	defaultOpenAIReasoningEffort = "low"
	defaultOpenAIMaxTokens       = 2000
)

// OpenAIRequest is a Chat Completions payload. Reasoning models (gpt-6-luna
// and other gpt-6* / gpt-5* / o-series) use max_completion_tokens rather than
// max_tokens. Temperature is only sent for non-reasoning models, or when
// reasoning_effort is "none".
type OpenAIRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenAIMessage `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
	ReasoningEffort     string          `json:"reasoning_effort,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []struct {
		Index        int           `json:"index"`
		Message      OpenAIMessage `json:"message"`
		LogProbs     interface{}   `json:"logprobs"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
}

func isReasoningModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.HasPrefix(m, "gpt-6") || strings.HasPrefix(m, "gpt-5") {
		return true
	}
	return isOSeriesModel(m)
}

func isOSeriesModel(m string) bool {
	for _, prefix := range []string{"o1", "o3", "o4"} {
		if m == prefix || strings.HasPrefix(m, prefix+"-") {
			return true
		}
	}
	return false
}

func normalizeReasoningEffort(s string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "" {
		v = defaultOpenAIReasoningEffort
	}
	switch v {
	case "none", "low", "medium", "high", "xhigh", "max":
		return v, nil
	default:
		return "", fmt.Errorf("invalid OPENAI_REASONING_EFFORT %q (allowed: none, low, medium, high, xhigh, max)", s)
	}
}

func sendOpenAITemperature(model, reasoningEffort string) bool {
	return !isReasoningModel(model) || strings.EqualFold(reasoningEffort, "none")
}

func buildOpenAIRequest(model string, temperature float64, maxTokens int, reasoningEffort, question string) OpenAIRequest {
	req := OpenAIRequest{
		Messages:            []OpenAIMessage{{Role: "user", Content: question}},
		Model:               model,
		MaxCompletionTokens: maxTokens,
	}
	if isReasoningModel(model) {
		req.ReasoningEffort = reasoningEffort
	}
	if sendOpenAITemperature(model, reasoningEffort) {
		req.Temperature = &temperature
	}
	return req
}

func AskOpenAI(openAIURL, openAIKey, openAIModel string, openAITemperature float64, openAIMaxTokens int, reasoningEffort, question string, verbose bool) (string, error) {
	// Start the spinner
	sp := newSpinner(os.Stderr, stdoutAndStderrAreTTY(), "Asking AI")
	sp.start()
	defer sp.stop()

	data := buildOpenAIRequest(openAIModel, openAITemperature, openAIMaxTokens, reasoningEffort, question)
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", openAIURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+openAIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-OK HTTP status from OpenAI: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if verbose {
		fmt.Printf("\nRaw response from OpenAI: %v", string(respBody)) // Debugging line to print raw response
	}

	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Choices) > 0 && apiResp.Choices[0].Message.Role == "assistant" {
		// Extract the content from the assistant's message
		//fmt.Println(apiResp.Choices[0].Message.Content)
		responseContent := strings.TrimSpace(apiResp.Choices[0].Message.Content)
		return responseContent, nil
	}

	return "", fmt.Errorf("no response from OpenAI")
}
