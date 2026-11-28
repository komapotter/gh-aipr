package main

import (
	"fmt"
	"strings"
	"testing"
)

const validPRJSON = `{"title":"Add test coverage","description":"Adds tests for the sanitizer.","changes":["Cover retry"],"testing":"go test ./..."}`

func TestGeneratePRDoesNotRetryWhenValid(t *testing.T) {
	calls := 0
	var got [][]OpenAIMessage
	complete := func(messages []OpenAIMessage) (string, error) {
		calls++
		got = append(got, messages)
		return validPRJSON, nil
	}
	title, body, err := generatePR(complete, "diff --git a/x", false)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if title != "Add test coverage" {
		t.Fatalf("title = %q", title)
	}
	if err := validateBody(body); err != nil {
		t.Fatal(err)
	}
	if len(got[0]) != 2 || got[0][0].Role != "system" || got[0][1].Role != "user" || got[0][1].Content != "diff --git a/x" {
		t.Fatalf("messages = %#v", got[0])
	}
	if strings.Contains(got[0][1].Content, "You generate") {
		t.Fatal("user message must be the diff only")
	}
}

func TestGeneratePRRetriesOnceOnInvalidOutput(t *testing.T) {
	calls := 0
	var got [][]OpenAIMessage
	complete := func(messages []OpenAIMessage) (string, error) {
		calls++
		got = append(got, messages)
		if calls == 1 {
			return `{"title":"","description":"Adds tests.","changes":["Add coverage"],"testing":"go test ./..."}`, nil
		}
		return validPRJSON, nil
	}
	title, body, err := generatePR(complete, "diff --git a/x", false)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if title != "Add test coverage" {
		t.Fatalf("title = %q", title)
	}
	if err := validateBody(body); err != nil {
		t.Fatal(err)
	}
	retry := got[1]
	if len(retry) != 4 {
		t.Fatalf("retry messages = %#v", retry)
	}
	if retry[0].Role != "system" || retry[1].Role != "user" || retry[1].Content != "diff --git a/x" {
		t.Fatalf("retry system/user = %#v", retry)
	}
	if retry[2].Role != "assistant" {
		t.Fatalf("retry missing previous assistant output: %#v", retry)
	}
	if retry[3].Role != "user" || !strings.Contains(retry[3].Content, "title is empty") {
		t.Fatalf("retry fix instruction = %#v", retry[3])
	}
}

func TestGeneratePRRetriesOnceOnInvalidJSON(t *testing.T) {
	calls := 0
	complete := func(messages []OpenAIMessage) (string, error) {
		calls++
		if calls == 1 {
			return "not json at all", nil
		}
		return validPRJSON, nil
	}
	title, _, err := generatePR(complete, "diff", true)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if title != "Add test coverage" {
		t.Fatalf("title = %q", title)
	}
}

func TestGeneratePRDoesNotRetryInfinitely(t *testing.T) {
	calls := 0
	complete := func(messages []OpenAIMessage) (string, error) {
		calls++
		return `{"title":"","description":"x","changes":["y"],"testing":"z"}`, nil
	}
	_, _, err := generatePR(complete, "diff", false)
	if err == nil {
		t.Fatal("expected validation error after retry")
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if !strings.Contains(err.Error(), "after retry") || !strings.Contains(err.Error(), "title is empty") {
		t.Fatalf("error = %v", err)
	}
}

func TestGeneratePRPropagatesCompleterErrorWithoutRetry(t *testing.T) {
	calls := 0
	complete := func(messages []OpenAIMessage) (string, error) {
		calls++
		return "", fmt.Errorf("network down")
	}
	_, _, err := generatePR(complete, "diff", false)
	if err == nil || err.Error() != "network down" {
		t.Fatalf("error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
