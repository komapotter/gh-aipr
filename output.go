package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// GitHub's pull request title limit.
const maxTitleLen = 256

var (
	remnantHR                = regexp.MustCompile(`^-{3,}$`)
	remnantEllipsis          = regexp.MustCompile(`^(\.{3,}|…)$`)
	remnantNumbered          = regexp.MustCompile(`(?i)^\d+\.\s*(change|変更点)\s*:\s*$`)
	remnantBulletPlaceholder = regexp.MustCompile(`(?i)^[-*]\s*(desc|説明)$`)
)

type prContent struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Changes     []string `json:"changes"`
	Testing     string   `json:"testing"`
}

func parsePRContent(raw string) (prContent, error) {
	cleaned := extractJSONObject(stripCodeFences(raw))
	var c prContent
	if err := json.Unmarshal([]byte(cleaned), &c); err != nil {
		return prContent{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return c, nil
}

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func assembleBody(c prContent) string {
	var b strings.Builder
	b.WriteString("## Description\n\n")
	if desc := strings.TrimSpace(c.Description); desc != "" {
		b.WriteString(desc)
		b.WriteString("\n")
	}
	b.WriteString("\n### Changes\n\n")
	n := 0
	for _, ch := range c.Changes {
		ch = strings.TrimSpace(ch)
		if ch == "" {
			continue
		}
		n++
		lines := strings.Split(ch, "\n")
		fmt.Fprintf(&b, "%d. %s\n", n, strings.TrimSpace(lines[0]))
		for _, extra := range lines[1:] {
			extra = strings.TrimSpace(extra)
			if extra == "" {
				continue
			}
			fmt.Fprintf(&b, "   %s\n", extra)
		}
	}
	b.WriteString("\n### Testing\n\n")
	if testing := strings.TrimSpace(c.Testing); testing != "" {
		b.WriteString(testing)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func sanitizeTitle(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	s = stripCodeFences(s)
	s = strings.TrimSpace(s)
	s = stripSurroundingQuotes(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func sanitizeBody(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	s = stripCodeFences(s)
	s = stripLeadingPreamble(s)
	s = stripCodeFences(s)
	s = stripTemplateRemnants(s)
	s = collapseBlankLines(s)
	return strings.TrimSpace(s)
}

func sanitizePRContent(c prContent) prContent {
	c.Title = sanitizeTitle(c.Title)
	c.Description = stripTemplateRemnants(strings.TrimSpace(c.Description))
	changes := make([]string, 0, len(c.Changes))
	for _, ch := range c.Changes {
		ch = strings.TrimSpace(ch)
		if ch == "" || isTemplateRemnantLine(ch) {
			continue
		}
		ch = stripTemplateRemnants(ch)
		if ch == "" {
			continue
		}
		changes = append(changes, ch)
	}
	c.Changes = changes
	c.Testing = stripTemplateRemnants(strings.TrimSpace(c.Testing))
	return c
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	rest := s
	if i := strings.Index(s, "\n"); i >= 0 {
		rest = s[i+1:]
	} else {
		return strings.Trim(s, "`")
	}
	rest = strings.TrimSpace(rest)
	if strings.HasSuffix(rest, "```") {
		rest = strings.TrimSpace(rest[:len(rest)-3])
	}
	return rest
}

func stripSurroundingQuotes(s string) string {
	s = strings.TrimSpace(s)
	pairs := [][2]string{
		{"\"", "\""},
		{"'", "'"},
		{"“", "”"},
		{"‘", "’"},
		{"「", "」"},
		{"『", "』"},
	}
	changed := true
	for changed {
		changed = false
		for _, p := range pairs {
			if len(s) >= len(p[0])+len(p[1]) && strings.HasPrefix(s, p[0]) && strings.HasSuffix(s, p[1]) {
				s = strings.TrimSpace(s[len(p[0]) : len(s)-len(p[1])])
				changed = true
				break
			}
		}
	}
	return s
}

func stripLeadingPreamble(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "## Description" || trim == "## 説明" {
			return strings.Join(lines[i:], "\n")
		}
	}
	return s
}

func stripTemplateRemnants(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if isTemplateRemnantLine(line) {
			continue
		}
		out = append(out, strings.TrimRight(line, " \t"))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func isFenceLine(line string) bool {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "```") {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(s, "```"))
	return !strings.ContainsAny(rest, " \t")
}

func isTemplateRemnantLine(line string) bool {
	s := strings.TrimSpace(line)
	if s == "" {
		return false
	}
	return isFenceLine(s) ||
		remnantHR.MatchString(s) ||
		remnantEllipsis.MatchString(s) ||
		remnantNumbered.MatchString(s) ||
		remnantBulletPlaceholder.MatchString(s)
}

func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

func validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title is empty")
	}
	if strings.ContainsAny(title, "\n\r") {
		return fmt.Errorf("title is multi-line")
	}
	n := utf8.RuneCountInString(title)
	if n > maxTitleLen {
		return fmt.Errorf("title is too long (%d runes, max %d)", n, maxTitleLen)
	}
	return nil
}

func validateBody(body string) error {
	if !hasHeading(body, "## Description") {
		return fmt.Errorf("body missing heading ## Description")
	}
	if !hasHeading(body, "### Changes") {
		return fmt.Errorf("body missing heading ### Changes")
	}
	if !hasHeading(body, "### Testing") {
		return fmt.Errorf("body missing heading ### Testing")
	}
	if remnant := findTemplateRemnant(body); remnant != "" {
		return fmt.Errorf("body contains template remnant %q", remnant)
	}
	return nil
}

func hasHeading(body, heading string) bool {
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == heading {
			return true
		}
	}
	return false
}

func findTemplateRemnant(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if isTemplateRemnantLine(line) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func parseAssembleSanitizeValidate(raw string) (string, string, error) {
	content, err := parsePRContent(raw)
	if err != nil {
		return "", "", err
	}
	content = sanitizePRContent(content)
	title := content.Title
	body := sanitizeBody(assembleBody(content))
	if err := validateTitle(title); err != nil {
		return title, body, err
	}
	if err := validateBody(body); err != nil {
		return title, body, err
	}
	return title, body, nil
}
