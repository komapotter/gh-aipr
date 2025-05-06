package main

import (
	"fmt"
	"time"
)

// Spinner represents a text-based spinner for command-line applications
type Spinner struct {
	chars     []rune
	message   string
	delay     time.Duration
	stopChan  chan struct{}
	dotDelay  time.Duration
	showDots  bool
	lastDot   time.Time
	dots      string
	isRunning bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		chars:    []rune(`|/-\`),
		message:  message,
		delay:    100 * time.Millisecond,
		stopChan: make(chan struct{}),
		dotDelay: time.Second,
		showDots: true,
		lastDot:  time.Now(),
		dots:     "",
	}
}

// WithDots enables or disables the dots after the message
func (s *Spinner) WithDots(enabled bool) *Spinner {
	s.showDots = enabled
	return s
}

// WithDelay sets the delay between spinner updates
func (s *Spinner) WithDelay(delay time.Duration) *Spinner {
	s.delay = delay
	return s
}

// Start starts the spinner
func (s *Spinner) Start() {
	if s.isRunning {
		return
	}
	s.isRunning = true
	s.stopChan = make(chan struct{})
	go s.run()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	if !s.isRunning {
		return
	}
	s.isRunning = false
	close(s.stopChan)
	fmt.Printf("\r\033[K") // Clear the entire line when done
}

// run is the goroutine that displays the spinner
func (s *Spinner) run() {
	i := 0
	for {
		select {
		case <-s.stopChan:
			return
		default:
			if s.showDots && time.Since(s.lastDot) >= s.dotDelay {
				s.dots += "."
				s.lastDot = time.Now()
			}
			
			fmt.Printf("\r %c %s%s", s.chars[i%len(s.chars)], s.message, s.dots)
			i++
			time.Sleep(s.delay)
		}
	}
}