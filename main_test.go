package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPrintHelpDocumentsVersionFlag(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed with error: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	printHelp()
	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading help output: %v", err)
	}
	r.Close()
	out := buf.String()
	if !strings.Contains(out, "--version") || !strings.Contains(out, "Print version and exit") {
		t.Fatalf("help missing --version documentation:\n%s", out)
	}
	if strings.Contains(out, " -V ") || strings.Contains(out, "  -V") {
		t.Fatalf("help should document --version, not -V:\n%s", out)
	}
	for _, cmd := range []string{"auth register", "auth remove", "auth status", "auth switch"} {
		if !strings.Contains(out, cmd) {
			t.Fatalf("help missing %s:\n%s", cmd, out)
		}
	}
	if !strings.Contains(out, "OPENAI_API_KEY") {
		t.Fatalf("help missing env documentation:\n%s", out)
	}
	if !strings.Contains(out, "gh aipr auth register") {
		t.Fatalf("help missing resolution fallback:\n%s", out)
	}
	if strings.Contains(out, "login") || strings.Contains(out, "logout") {
		t.Fatalf("help should not mention login/logout:\n%s", out)
	}
	if !strings.Contains(out, "gpt-6-luna") {
		t.Fatalf("help missing gpt-6-luna default:\n%s", out)
	}
	if strings.Contains(out, "gpt-4o") {
		t.Fatalf("help should not document gpt-4o as the default:\n%s", out)
	}
	if !strings.Contains(out, "OPENAI_REASONING_EFFORT") {
		t.Fatalf("help missing OPENAI_REASONING_EFFORT:\n%s", out)
	}
	if !strings.Contains(out, "default: 2000") {
		t.Fatalf("help missing OPENAI_MAX_TOKENS default 2000:\n%s", out)
	}
}

func TestConfigDefaultOpenAISettings(t *testing.T) {
	if defaultOpenAIModel != "gpt-6-luna" {
		t.Fatalf("defaultOpenAIModel = %q, want gpt-6-luna", defaultOpenAIModel)
	}
	if defaultOpenAIReasoningEffort != "low" {
		t.Fatalf("defaultOpenAIReasoningEffort = %q, want low", defaultOpenAIReasoningEffort)
	}
	if defaultOpenAIMaxTokens != 2000 {
		t.Fatalf("defaultOpenAIMaxTokens = %d, want 2000", defaultOpenAIMaxTokens)
	}

	typ := reflect.TypeOf(Config{})
	want := map[string]string{
		"OpenAIModel":           "gpt-6-luna",
		"OpenAIMaxTokens":       "2000",
		"OpenAIReasoningEffort": "low",
	}
	for field, def := range want {
		f, ok := typ.FieldByName(field)
		if !ok {
			t.Fatalf("Config.%s missing", field)
		}
		if got := f.Tag.Get("default"); got != def {
			t.Fatalf("Config.%s default tag = %q, want %s", field, got, def)
		}
	}
}

func TestVersionFlagParsesAsVersion(t *testing.T) {
	parse := func(args ...string) (verbose, create, help, titleOnly, bodyOnly, japanise, version bool, issueNo int) {
		fs := flag.NewFlagSet("aipr", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		registerAppFlags(fs, &verbose, &create, &help, &titleOnly, &bodyOnly, &japanise, &version, &issueNo)
		if err := fs.Parse(args); err != nil {
			t.Fatalf("Parse(%q) error: %v", args, err)
		}
		return
	}

	verbose, _, _, _, _, _, version, _ := parse("--version")
	if !version {
		t.Fatal("--version should set the version flag")
	}
	if verbose {
		t.Fatal("--version must not set verbose")
	}

	verbose, _, _, _, _, _, version, _ = parse("--verbose")
	if !verbose {
		t.Fatal("--verbose should set verbose")
	}
	if version {
		t.Fatal("--verbose must not set the version flag")
	}

	verbose, create, help, titleOnly, bodyOnly, japanise, version, issueNo := parse("--verbose", "--version", "--issue-no", "22")
	if !verbose || create || help || titleOnly || bodyOnly || japanise || !version || issueNo != 22 {
		t.Fatalf("got verbose=%v create=%v help=%v title=%v body=%v japanise=%v version=%v issueNo=%d",
			verbose, create, help, titleOnly, bodyOnly, japanise, version, issueNo)
	}
}
