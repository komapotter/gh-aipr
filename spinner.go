package main

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

const ghSpinnerInterval = 120 * time.Millisecond

type spinnerCtl struct {
	inner   *spinner.Spinner
	enabled bool
	once    sync.Once
}

func spinnerColor() string {
	if os.Getenv("NO_COLOR") != "" {
		return ""
	}
	return "fgCyan"
}

func newSpinner(w io.Writer, enabled bool, message string) *spinnerCtl {
	opts := []spinner.Option{}
	if color := spinnerColor(); color != "" {
		opts = append(opts, spinner.WithColor(color))
	}
	if f, ok := w.(*os.File); ok {
		opts = append(opts, spinner.WithWriterFile(f))
	} else {
		opts = append(opts, spinner.WithWriter(w))
	}

	// Same set and interval as GitHub CLI (cli/cli uses CharSets[11] @ 120ms).
	inner := spinner.New(spinner.CharSets[11], ghSpinnerInterval, opts...)
	// ⣾ Getting git diff
	inner.Suffix = " " + message
	inner.HideCursor = true
	return &spinnerCtl{inner: inner, enabled: enabled}
}

func stdoutAndStderrAreTTY() bool {
	return isCharDevice(os.Stdout) && isCharDevice(os.Stderr)
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func (s *spinnerCtl) start() {
	if !s.enabled {
		return
	}
	s.inner.Start()
}

func (s *spinnerCtl) stop() {
	s.once.Do(func() {
		s.inner.Stop()
	})
}
