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

// OpenAIRequest is a Chat Completions payload. gpt-6-luna is a reasoning
// model, so the request uses max_completion_tokens (not max_tokens).
// Temperature is omitted for gpt-6-luna because it is only valid when
// reasoning_effort is "none"; the model default is medium.
type OpenAIRequest struct {
	Model               string          `json:"model"`
	Messages            []OpenAIMessage `json:"messages"`
	MaxCompletionTokens int             `json:"max_completion_tokens,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
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

func isGPT6Luna(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return m == defaultOpenAIModel || strings.HasPrefix(m, defaultOpenAIModel+"-")
}

func buildOpenAIRequest(model string, temperature float64, maxTokens int, question string) OpenAIRequest {
	req := OpenAIRequest{
		Messages:            []OpenAIMessage{{Role: "user", Content: question}},
		Model:               model,
		MaxCompletionTokens: maxTokens,
	}
	// gpt-6-luna only accepts temperature when reasoning_effort is "none".
	// Keep sending it for other models (for example gpt-4o via OPENAI_MODEL).
	if !isGPT6Luna(model) {
		req.Temperature = &temperature
	}
	return req
}

func AskOpenAI(openAIURL, openAIKey, openAIModel string, openAITemperature float64, openAIMaxTokens int, question string, verbose bool) (string, error) {
	// Start the spinner
	sp := newSpinner(os.Stderr, stdoutAndStderrAreTTY(), "Asking AI")
	sp.start()
	defer sp.stop()

	data := buildOpenAIRequest(openAIModel, openAITemperature, openAIMaxTokens, question)
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
