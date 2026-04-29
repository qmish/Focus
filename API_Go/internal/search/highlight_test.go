package search

import (
	"strings"
	"testing"
)

func TestHighlightSnippet(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		query    string
		maxLen   int
		contains []string
		notHas   []string
	}{
		{
			name:     "simple match",
			content:  "Hello world, this is a Focus message.",
			query:    "world",
			maxLen:   100,
			contains: []string{"<mark>world</mark>", "Focus message"},
		},
		{
			name:     "case insensitive but case preserving",
			content:  "Привет, это Сообщение",
			query:    "сообщение",
			maxLen:   100,
			contains: []string{"<mark>Сообщение</mark>"},
		},
		{
			name:     "html escaped",
			content:  "<script>alert(1)</script> hello world",
			query:    "alert",
			maxLen:   100,
			contains: []string{"&lt;script&gt;", "<mark>alert</mark>"},
			notHas:   []string{"<script>"},
		},
		{
			name:     "long content trimmed around match",
			content:  strings.Repeat("a ", 200) + "needle " + strings.Repeat("b ", 200),
			query:    "needle",
			maxLen:   60,
			contains: []string{"<mark>needle</mark>", "..."},
		},
		{
			name:     "no match returns truncated",
			content:  "There is no match here at all in this body of text",
			query:    "xyzzy",
			maxLen:   20,
			contains: []string{"There is no match"},
			notHas:   []string{"<mark>"},
		},
		{
			name:     "empty query returns truncated content",
			content:  "Hello world",
			query:    "",
			maxLen:   100,
			contains: []string{"Hello world"},
			notHas:   []string{"<mark>"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := HighlightSnippet(tc.content, tc.query, tc.maxLen)
			for _, s := range tc.contains {
				if !strings.Contains(out, s) {
					t.Errorf("expected output to contain %q, got %q", s, out)
				}
			}
			for _, s := range tc.notHas {
				if strings.Contains(out, s) {
					t.Errorf("expected output NOT to contain %q, got %q", s, out)
				}
			}
		})
	}
}

func TestHighlightSnippet_EmptyContent(t *testing.T) {
	if got := HighlightSnippet("", "x", 100); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
