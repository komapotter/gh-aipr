package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func marshalRequest(t *testing.T, req OpenAIRequest) map[string]any {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestBuildOpenAIRequestDefaultLuna(t *testing.T) {
	got := marshalRequest(t, buildOpenAIRequest(defaultOpenAIModel, 0.1, defaultOpenAIMaxTokens, defaultOpenAIReasoningEffort, "hello"))
	if got["model"] != defaultOpenAIModel {
		t.Fatalf("model = %v, want %s", got["model"], defaultOpenAIModel)
	}
	if got["reasoning_effort"] != "low" {
		t.Fatalf("reasoning_effort = %v, want low", got["reasoning_effort"])
	}
	if _, ok := got["temperature"]; ok {
		t.Fatalf("default gpt-6-luna payload must omit temperature: %v", got)
	}
	if _, ok := got["max_tokens"]; ok {
		t.Fatalf("payload must omit max_tokens: %v", got)
	}
	if got["max_completion_tokens"] != float64(2000) {
		t.Fatalf("max_completion_tokens = %v, want 2000", got["max_completion_tokens"])
	}
}

func TestBuildOpenAIRequestLunaNoneSendsTemperature(t *testing.T) {
	got := marshalRequest(t, buildOpenAIRequest(defaultOpenAIModel, 0.1, defaultOpenAIMaxTokens, "none", "hello"))
	if got["reasoning_effort"] != "none" {
		t.Fatalf("reasoning_effort = %v, want none", got["reasoning_effort"])
	}
	if got["temperature"] != 0.1 {
		t.Fatalf("temperature = %v, want 0.1 when reasoning_effort is none", got["temperature"])
	}
}

func TestBuildOpenAIRequestGPT4oOmitsReasoningEffort(t *testing.T) {
	got := marshalRequest(t, buildOpenAIRequest("gpt-4o", 0.1, defaultOpenAIMaxTokens, "low", "hello"))
	if _, ok := got["reasoning_effort"]; ok {
		t.Fatalf("gpt-4o payload must omit reasoning_effort: %v", got)
	}
	if got["temperature"] != 0.1 {
		t.Fatalf("temperature = %v, want 0.1", got["temperature"])
	}
	if got["max_completion_tokens"] != float64(2000) {
		t.Fatalf("max_completion_tokens = %v, want 2000", got["max_completion_tokens"])
	}
}

func TestNormalizeReasoningEffortInvalid(t *testing.T) {
	_, err := normalizeReasoningEffort("turbo")
	if err == nil {
		t.Fatal("expected error for invalid OPENAI_REASONING_EFFORT")
	}
	if !strings.Contains(err.Error(), "OPENAI_REASONING_EFFORT") || !strings.Contains(err.Error(), "turbo") {
		t.Fatalf("error = %v", err)
	}

	got, err := normalizeReasoningEffort("HIGH")
	if err != nil {
		t.Fatal(err)
	}
	if got != "high" {
		t.Fatalf("normalizeReasoningEffort(HIGH) = %q, want high", got)
	}

	got, err = normalizeReasoningEffort("")
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultOpenAIReasoningEffort {
		t.Fatalf("empty effort = %q, want %s", got, defaultOpenAIReasoningEffort)
	}
}

func TestLoadAppConfigRejectsInvalidReasoningEffort(t *testing.T) {
	t.Setenv("OPENAI_REASONING_EFFORT", "fast")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	_, err := loadAppConfig()
	if err == nil {
		t.Fatal("expected loadAppConfig to reject invalid OPENAI_REASONING_EFFORT")
	}
	if !strings.Contains(err.Error(), "OPENAI_REASONING_EFFORT") || !strings.Contains(err.Error(), "fast") {
		t.Fatalf("error = %v", err)
	}
}

func TestIsReasoningModel(t *testing.T) {
	for _, model := range []string{"gpt-6-luna", "GPT-6-LUNA", "gpt-6.2-astra", "gpt-5.4", "o1", "o3-mini", "o4-mini"} {
		if !isReasoningModel(model) {
			t.Fatalf("%s should be a reasoning model", model)
		}
	}
	for _, model := range []string{"gpt-4o", "gpt-4.1", "claude-3-haiku-20240307"} {
		if isReasoningModel(model) {
			t.Fatalf("%s should not be a reasoning model", model)
		}
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

	got, err := AskOpenAI(srv.URL, "test-key", defaultOpenAIModel, 0.1, defaultOpenAIMaxTokens, defaultOpenAIReasoningEffort, []OpenAIMessage{{Role: "user", Content: "summarize"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Add tests" {
		t.Fatalf("response = %q, want trimmed assistant content", got)
	}
	if !strings.Contains(string(gotBody), `"model":"gpt-6-luna"`) {
		t.Fatalf("request body missing gpt-6-luna: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), `"reasoning_effort":"low"`) {
		t.Fatalf("request body missing reasoning_effort low: %s", gotBody)
	}
	if strings.Contains(string(gotBody), `"temperature"`) {
		t.Fatalf("request body must omit temperature: %s", gotBody)
	}
	if strings.Contains(string(gotBody), `"max_tokens"`) {
		t.Fatalf("request body must omit max_tokens: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), `"max_completion_tokens":2000`) {
		t.Fatalf("request body missing max_completion_tokens 2000: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), `"type":"json_schema"`) {
		t.Fatalf("request body missing json_schema response_format: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), `"strict":true`) {
		t.Fatalf("request body missing strict json_schema: %s", gotBody)
	}
}

func TestBuildOpenAIChatRequestJSONSchema(t *testing.T) {
	got := marshalRequest(t, buildOpenAIChatRequest(defaultOpenAIModel, 0.1, defaultOpenAIMaxTokens, defaultOpenAIReasoningEffort,
		[]OpenAIMessage{{Role: "system", Content: "contract"}, {Role: "user", Content: "diff"}}, jsonSchemaFormat()))
	format, ok := got["response_format"].(map[string]any)
	if !ok {
		t.Fatalf("response_format missing: %v", got)
	}
	if format["type"] != "json_schema" {
		t.Fatalf("response_format.type = %v", format["type"])
	}
	schema, ok := format["json_schema"].(map[string]any)
	if !ok {
		t.Fatalf("json_schema missing: %v", format)
	}
	if schema["name"] != "pull_request" {
		t.Fatalf("schema name = %v", schema["name"])
	}
	if schema["strict"] != true {
		t.Fatalf("strict = %v", schema["strict"])
	}
	msgs, ok := got["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("messages = %v", got["messages"])
	}
}

func TestAskOpenAIFallsBackToJSONObject(t *testing.T) {
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
		}
		bodies = append(bodies, body)
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"json_schema is not supported"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"{\"title\":\"Ok\"}"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	got, err := AskOpenAI(srv.URL, "test-key", defaultOpenAIModel, 0.1, defaultOpenAIMaxTokens, defaultOpenAIReasoningEffort, []OpenAIMessage{{Role: "user", Content: "diff"}}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"title":"Ok"}` {
		t.Fatalf("response = %q", got)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d, want 2", len(bodies))
	}
	if !strings.Contains(string(bodies[0]), `"type":"json_schema"`) {
		t.Fatalf("first request should use json_schema: %s", bodies[0])
	}
	if !strings.Contains(string(bodies[1]), `"type":"json_object"`) {
		t.Fatalf("fallback request should use json_object: %s", bodies[1])
	}
	if strings.Contains(string(bodies[1]), `"json_schema"`) {
		t.Fatalf("json_object fallback must omit json_schema: %s", bodies[1])
	}
}

func TestAskOpenAIDoesNotFallbackOnUnauthorized(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid_api_key"}}`))
	}))
	defer srv.Close()

	_, err := AskOpenAI(srv.URL, "bad-key", defaultOpenAIModel, 0.1, defaultOpenAIMaxTokens, defaultOpenAIReasoningEffort, []OpenAIMessage{{Role: "user", Content: "diff"}}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "Unauthorized") {
		t.Fatalf("error = %v", err)
	}
}
