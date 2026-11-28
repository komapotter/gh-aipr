package main

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestAssembleBodyMatchesRequiredHeadings(t *testing.T) {
	got := assembleBody(prContent{
		Description: "Adds a sanitizer.",
		Changes:     []string{"Strip fences", "Validate headings\nKeep nested detail"},
		Testing:     "go test ./...",
	})
	want := strings.TrimSpace(`
## Description

Adds a sanitizer.

### Changes

1. Strip fences
2. Validate headings
   Keep nested detail

### Testing

go test ./...
`)
	if got != want {
		t.Fatalf("assembleBody mismatch\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestAssembleBodyKeepsEmptySectionsHeadings(t *testing.T) {
	got := assembleBody(prContent{})
	for _, heading := range []string{"## Description", "### Changes", "### Testing"} {
		if !hasHeading(got, heading) {
			t.Fatalf("missing %s in:\n%s", heading, got)
		}
	}
}

func TestSanitizeTitleStripsQuotesAndCollapsesLines(t *testing.T) {
	got := sanitizeTitle("  \"Fix the\n  spinner\"  ")
	if got != "Fix the spinner" {
		t.Fatalf("sanitizeTitle = %q", got)
	}
	got = sanitizeTitle("「日本語タイトル」")
	if got != "日本語タイトル" {
		t.Fatalf("sanitizeTitle japanese quotes = %q", got)
	}
	got = sanitizeTitle("```\nAdd tests\n```")
	if got != "Add tests" {
		t.Fatalf("sanitizeTitle fences = %q", got)
	}
}

func TestSanitizeBodyStripsPreambleRemnantsAndEmptySections(t *testing.T) {
	in := "Sure, here is the PR:\n\n```markdown\n---\n## Description\n\n* desc\n\nReal summary.\n\n### Changes\n\n1. change:\n1. Add sanitizer\n...\n...\n\n### Testing\n\n- desc\n\ngo test ./...\n```\n"
	got := sanitizeBody(in)
	if strings.Contains(got, "Sure, here is the PR") {
		t.Fatalf("preamble not stripped:\n%s", got)
	}
	if strings.Contains(got, "```") {
		t.Fatalf("code fences not stripped:\n%s", got)
	}
	if remnant := findTemplateRemnant(got); remnant != "" {
		t.Fatalf("remnant %q left in:\n%s", remnant, got)
	}
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("empty sections not collapsed:\n%s", got)
	}
	if !strings.Contains(got, "Real summary.") || !strings.Contains(got, "1. Add sanitizer") || !strings.Contains(got, "go test ./...") {
		t.Fatalf("real content was stripped:\n%s", got)
	}
}

func TestSanitizePRContentDropsPlaceholderChanges(t *testing.T) {
	got := sanitizePRContent(prContent{
		Title:       "\"Add output sanitizer\"",
		Description: "* desc\nKeeps the real summary.\n...",
		Changes:     []string{"1. change:", "Strip remnants", ""},
		Testing:     "- desc\ngo test ./...",
	})
	if got.Title != "Add output sanitizer" {
		t.Fatalf("title = %q", got.Title)
	}
	if strings.Contains(got.Description, "* desc") || strings.Contains(got.Description, "...") {
		t.Fatalf("description remnants remain: %q", got.Description)
	}
	if !strings.Contains(got.Description, "Keeps the real summary.") {
		t.Fatalf("description lost real text: %q", got.Description)
	}
	if len(got.Changes) != 1 || got.Changes[0] != "Strip remnants" {
		t.Fatalf("changes = %#v", got.Changes)
	}
	if strings.Contains(got.Testing, "- desc") || !strings.Contains(got.Testing, "go test ./...") {
		t.Fatalf("testing = %q", got.Testing)
	}
}

func TestValidateTitle(t *testing.T) {
	if err := validateTitle("Add sanitizer"); err != nil {
		t.Fatalf("valid title rejected: %v", err)
	}
	if err := validateTitle("  "); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty title error = %v", err)
	}
	if err := validateTitle("one\ntwo"); err == nil || !strings.Contains(err.Error(), "multi-line") {
		t.Fatalf("multi-line title error = %v", err)
	}
	long := strings.Repeat("あ", maxTitleLen+1)
	if utf8.RuneCountInString(long) != maxTitleLen+1 {
		t.Fatal("test setup")
	}
	if err := validateTitle(long); err == nil || !strings.Contains(err.Error(), "too long") {
		t.Fatalf("too-long title error = %v", err)
	}
	if err := validateTitle(strings.Repeat("a", maxTitleLen)); err != nil {
		t.Fatalf("max-length title rejected: %v", err)
	}
}

func TestValidateBody(t *testing.T) {
	valid := assembleBody(prContent{
		Description: "Summary.",
		Changes:     []string{"Do the thing"},
		Testing:     "go test ./...",
	})
	if err := validateBody(valid); err != nil {
		t.Fatalf("valid body rejected: %v", err)
	}
	if err := validateBody("### Changes\n\n1. x\n\n### Testing\n\ny"); err == nil || !strings.Contains(err.Error(), "## Description") {
		t.Fatalf("missing Description error = %v", err)
	}
	if err := validateBody("## Description\n\nx\n\n### Testing\n\ny"); err == nil || !strings.Contains(err.Error(), "### Changes") {
		t.Fatalf("missing Changes error = %v", err)
	}
	if err := validateBody("## Description\n\nx\n\n### Changes\n\n1. y"); err == nil || !strings.Contains(err.Error(), "### Testing") {
		t.Fatalf("missing Testing error = %v", err)
	}
	withRemnant := valid + "\n\n---\n"
	if err := validateBody(withRemnant); err == nil || !strings.Contains(err.Error(), "template remnant") {
		t.Fatalf("remnant error = %v", err)
	}
}

func TestParsePRContentStripsFencesAndPreamble(t *testing.T) {
	raw := "Here you go:\n```json\n{\"title\":\"Hello\",\"description\":\"D\",\"changes\":[\"C\"],\"testing\":\"T\"}\n```"
	got, err := parsePRContent(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Hello" || got.Description != "D" || got.Testing != "T" || len(got.Changes) != 1 || got.Changes[0] != "C" {
		t.Fatalf("parsed = %#v", got)
	}
}

func TestParseAssembleSanitizeValidateRoundTrip(t *testing.T) {
	raw := "```json\n{\"title\":\"  \\\"Fix output\\\"  \",\"description\":\"Stabilizes PR markdown.\",\"changes\":[\"Assemble from JSON\"],\"testing\":\"go test ./...\"}\n```"
	title, body, err := parseAssembleSanitizeValidate(raw)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Fix output" {
		t.Fatalf("title = %q", title)
	}
	if err := validateBody(body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "## Description") || !strings.Contains(body, "Stabilizes PR markdown.") {
		t.Fatalf("body = %q", body)
	}
}

func TestPROutputSchemaIsValidJSON(t *testing.T) {
	var v any
	if err := json.Unmarshal([]byte(prOutputSchema), &v); err != nil {
		t.Fatal(err)
	}
}

func TestSystemPromptHasConstraintsNotSampleTemplate(t *testing.T) {
	for _, japanise := range []bool{false, true} {
		p := systemPrompt(japanise)
		if !strings.Contains(p, "## Description") || !strings.Contains(p, "### Changes") || !strings.Contains(p, "### Testing") {
			t.Fatalf("prompt missing required headings (japanise=%v):\n%s", japanise, p)
		}
		if strings.Contains(p, "Here is a sample") || strings.Contains(p, "サンプルです") {
			t.Fatalf("prompt still contains sample template (japanise=%v):\n%s", japanise, p)
		}
		if strings.Contains(p, "1. change:") || strings.Contains(p, "1. 変更点:") {
			t.Fatalf("prompt still contains fill-in change template (japanise=%v):\n%s", japanise, p)
		}
	}
}
