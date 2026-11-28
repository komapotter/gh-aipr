package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

// prOutputSchema is the strict JSON Schema for structured PR title/body output.
const prOutputSchema = `{
  "type": "object",
  "properties": {
    "title": {
      "type": "string",
      "description": "Single-line pull request title without surrounding quotes"
    },
    "description": {
      "type": "string",
      "description": "Short summary of the change. No placeholders."
    },
    "changes": {
      "type": "array",
      "description": "Concrete change items, one per array element",
      "items": {
        "type": "string"
      }
    },
    "testing": {
      "type": "string",
      "description": "How to verify the change"
    }
  },
  "required": ["title", "description", "changes", "testing"],
  "additionalProperties": false
}`

// OpenAIRequest is a Chat Completions payload. Reasoning models (gpt-6-luna
// and other gpt-6* / gpt-5* / o-series) use max_completion_tokens rather than
// max_tokens. Temperature is only sent for non-reasoning models, or when
// reasoning_effort is "none".
type OpenAIRequest struct {
	Model               string                `json:"model"`
	Messages            []OpenAIMessage       `json:"messages"`
	MaxCompletionTokens int                   `json:"max_completion_tokens,omitempty"`
	Temperature         *float64              `json:"temperature,omitempty"`
	ReasoningEffort     string                `json:"reasoning_effort,omitempty"`
	ResponseFormat      *OpenAIResponseFormat `json:"response_format,omitempty"`
}

type OpenAIResponseFormat struct {
	Type       string            `json:"type"`
	JSONSchema *OpenAIJSONSchema `json:"json_schema,omitempty"`
}

type OpenAIJSONSchema struct {
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
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

type openAIStatusError struct {
	Status     string
	StatusCode int
	Body       string
}

func (e *openAIStatusError) Error() string {
	msg := fmt.Sprintf("received non-OK HTTP status from OpenAI: %s", e.Status)
	if strings.TrimSpace(e.Body) != "" {
		body := strings.TrimSpace(e.Body)
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		msg += ": " + body
	}
	return msg
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

func jsonSchemaFormat() *OpenAIResponseFormat {
	return &OpenAIResponseFormat{
		Type: "json_schema",
		JSONSchema: &OpenAIJSONSchema{
			Name:   "pull_request",
			Strict: true,
			Schema: json.RawMessage(prOutputSchema),
		},
	}
}

func jsonObjectFormat() *OpenAIResponseFormat {
	return &OpenAIResponseFormat{Type: "json_object"}
}

func buildOpenAIRequest(model string, temperature float64, maxTokens int, reasoningEffort, question string) OpenAIRequest {
	return buildOpenAIChatRequest(model, temperature, maxTokens, reasoningEffort,
		[]OpenAIMessage{{Role: "user", Content: question}}, nil)
}

func buildOpenAIChatRequest(model string, temperature float64, maxTokens int, reasoningEffort string, messages []OpenAIMessage, responseFormat *OpenAIResponseFormat) OpenAIRequest {
	req := OpenAIRequest{
		Messages:            messages,
		Model:               model,
		MaxCompletionTokens: maxTokens,
		ResponseFormat:      responseFormat,
	}
	if isReasoningModel(model) {
		req.ReasoningEffort = reasoningEffort
	}
	if sendOpenAITemperature(model, reasoningEffort) {
		req.Temperature = &temperature
	}
	return req
}

func AskOpenAI(openAIURL, openAIKey, openAIModel string, openAITemperature float64, openAIMaxTokens int, reasoningEffort string, messages []OpenAIMessage, verbose bool) (string, error) {
	// Start the spinner
	sp := newSpinner(os.Stderr, stdoutAndStderrAreTTY(), "Asking AI")
	sp.start()
	defer sp.stop()

	content, err := postOpenAIChat(openAIURL, openAIKey, buildOpenAIChatRequest(openAIModel, openAITemperature, openAIMaxTokens, reasoningEffort, messages, jsonSchemaFormat()), verbose)
	if err != nil {
		var se *openAIStatusError
		if errors.As(err, &se) && se.StatusCode == http.StatusBadRequest {
			return postOpenAIChat(openAIURL, openAIKey, buildOpenAIChatRequest(openAIModel, openAITemperature, openAIMaxTokens, reasoningEffort, messages, jsonObjectFormat()), verbose)
		}
		return "", err
	}
	return content, nil
}

func postOpenAIChat(openAIURL, openAIKey string, data OpenAIRequest, verbose bool) (string, error) {
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if verbose {
		fmt.Printf("\nRaw response from OpenAI: %v", string(respBody))
	}

	if resp.StatusCode != http.StatusOK {
		return "", &openAIStatusError{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Choices) > 0 && apiResp.Choices[0].Message.Role == "assistant" {
		responseContent := strings.TrimSpace(apiResp.Choices[0].Message.Content)
		return responseContent, nil
	}

	return "", fmt.Errorf("no response from OpenAI")
}
