package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildOpenAIRequestGPT6LunaOmitsUnsupportedParams(t *testing.T) {
	req := buildOpenAIRequest(defaultOpenAIModel, 0.1, 450, "hello")
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got["model"] != defaultOpenAIModel {
		t.Fatalf("model = %v, want %s", got["model"], defaultOpenAIModel)
	}
	if _, ok := got["temperature"]; ok {
		t.Fatalf("gpt-6-luna payload must omit temperature: %s", payload)
	}
	if _, ok := got["max_tokens"]; ok {
		t.Fatalf("gpt-6-luna payload must omit max_tokens: %s", payload)
	}
	if got["max_completion_tokens"] != float64(450) {
		t.Fatalf("max_completion_tokens = %v, want 450", got["max_completion_tokens"])
	}
	if _, ok := got["reasoning_effort"]; ok {
		t.Fatalf("payload should leave reasoning_effort at the model default: %s", payload)
	}
}

func TestBuildOpenAIRequestLegacyModelKeepsTemperature(t *testing.T) {
	req := buildOpenAIRequest("gpt-4o", 0.1, 450, "hello")
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got["temperature"] != 0.1 {
		t.Fatalf("temperature = %v, want 0.1", got["temperature"])
	}
	if _, ok := got["max_tokens"]; ok {
		t.Fatalf("payload must use max_completion_tokens, not max_tokens: %s", payload)
	}
	if got["max_completion_tokens"] != float64(450) {
		t.Fatalf("max_completion_tokens = %v, want 450", got["max_completion_tokens"])
	}
}

func TestIsGPT6Luna(t *testing.T) {
	if !isGPT6Luna("gpt-6-luna") {
		t.Fatal("gpt-6-luna should be detected")
	}
	if !isGPT6Luna("GPT-6-LUNA") {
		t.Fatal("model id match should be case-insensitive")
	}
	if isGPT6Luna("gpt-4o") {
		t.Fatal("gpt-4o is not gpt-6-luna")
	}
}

func TestAskOpenAIUsesChatCompletionsPayload(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
		}
		gotBody = body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":" Add tests "},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	got, err := AskOpenAI(srv.URL, "test-key", defaultOpenAIModel, 0.1, 450, "summarize", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Add tests" {
		t.Fatalf("response = %q, want trimmed assistant content", got)
	}
	if !strings.Contains(string(gotBody), `"model":"gpt-6-luna"`) {
		t.Fatalf("request body missing gpt-6-luna: %s", gotBody)
	}
	if strings.Contains(string(gotBody), `"temperature"`) {
		t.Fatalf("request body must omit temperature: %s", gotBody)
	}
	if strings.Contains(string(gotBody), `"max_tokens"`) {
		t.Fatalf("request body must omit max_tokens: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), `"max_completion_tokens":450`) {
		t.Fatalf("request body missing max_completion_tokens: %s", gotBody)
	}
}
