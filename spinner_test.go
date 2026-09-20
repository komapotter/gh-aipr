package main

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/briandowns/spinner"
)

func TestSpinnerUsesBriandownsCharSet11(t *testing.T) {
	want := []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	if !equalSlices(spinner.CharSets[11], want) {
		t.Fatalf("spinner.CharSets[11] = %q, want %q", spinner.CharSets[11], want)
	}
	sp := newSpinner(io.Discard, true, "Getting git diff")
	if sp.inner.Delay != ghSpinnerInterval {
		t.Fatalf("Delay = %v, want %v", sp.inner.Delay, ghSpinnerInterval)
	}
	if !sp.inner.HideCursor {
		t.Fatal("HideCursor = false, want true")
	}
}

func TestSpinnerFixedWidthFrames(t *testing.T) {
	for i, frame := range spinner.CharSets[11] {
		if n := utf8.RuneCountInString(frame); n != 1 {
			t.Fatalf("CharSets[11][%d] = %q has width %d, want 1", i, frame, n)
		}
	}
}

func TestSpinnerLayoutIsFrameThenLabel(t *testing.T) {
	const message = "Getting git diff"
	sp := newSpinner(io.Discard, true, message)
	if sp.inner.Prefix != "" {
		t.Fatalf("Prefix = %q, want empty so the braille glyph comes first", sp.inner.Prefix)
	}
	if sp.inner.Suffix != " "+message {
		t.Fatalf("Suffix = %q, want %q", sp.inner.Suffix, " "+message)
	}
}

func TestSpinnerStopIdempotent(t *testing.T) {
	sp := newSpinner(io.Discard, true, "Getting git diff")
	sp.start()
	time.Sleep(15 * time.Millisecond)
	sp.stop()
	sp.stop()

	idle := newSpinner(io.Discard, true, "Getting git diff")
	idle.stop()
}

func TestSpinnerNoAnimationWhenDisabled(t *testing.T) {
	var buf bytes.Buffer
	sp := newSpinner(&buf, false, "Getting git diff")
	sp.start()
	time.Sleep(20 * time.Millisecond)
	sp.stop()
	if buf.Len() != 0 {
		t.Fatalf("disabled spinner wrote %q", buf.String())
	}
}

func TestIsCharDevicePipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()
	if isCharDevice(w) || isCharDevice(r) {
		t.Fatal("pipe should not be treated as a TTY")
	}
}

func TestSpinnerSkipsColorWhenNoColorSet(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := spinnerColor(); got != "" {
		t.Fatalf("spinnerColor() = %q, want empty when NO_COLOR is set", got)
	}
}

func TestSpinnerUsesCyanWhenColorAllowed(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if got := spinnerColor(); got != "fgCyan" {
		t.Fatalf("spinnerColor() = %q, want fgCyan", got)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
