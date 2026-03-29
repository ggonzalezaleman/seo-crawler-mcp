package parser

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestCountWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello world", 2},
		{"  spaced   out  ", 2},
		{"one", 1},
		{"the quick brown fox jumps over the lazy dog", 9},
	}
	for _, tt := range tests {
		got := CountWords(tt.input)
		if got != tt.want {
			t.Errorf("CountWords(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestMainContentWordCount_WithMain(t *testing.T) {
	html := `<html><body>
		<header>Header stuff here</header>
		<main>Main content with several words in it</main>
		<footer>Footer stuff</footer>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}

	text := ExtractMainContentText(doc)
	count := CountWords(text)
	if count != 7 {
		t.Errorf("main content word count = %d, want 7 (got text: %q)", count, text)
	}
}

func TestMainContentWordCount_Fallback(t *testing.T) {
	html := `<html><body>
		<header>Header words here</header>
		<nav>Nav words</nav>
		<div>Body content words here in div</div>
		<footer>Footer words</footer>
		<aside>Aside words</aside>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}

	text := ExtractMainContentText(doc)
	count := CountWords(text)
	// Should only include "Body content words here in div" = 6 words.
	if count != 6 {
		t.Errorf("fallback word count = %d, want 6 (got text: %q)", count, text)
	}
}

func TestMainContentWordCount_DoesNotCountScriptOrNoscript(t *testing.T) {
	html := `<html><body>
		<div>Visible body copy here</div>
		<script>console.log('secret script words')</script>
		<noscript>Fallback noscript copy should not count</noscript>
		<style>.foo { content: 'style words'; }</style>
	</body></html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}

	allText := ExtractVisibleText(doc)
	mainText := ExtractMainContentText(doc)
	allCount := CountWords(allText)
	mainCount := CountWords(mainText)

	if allCount != 4 {
		t.Fatalf("visible word count = %d, want 4 (got text: %q)", allCount, allText)
	}
	if mainCount != 4 {
		t.Fatalf("main content word count = %d, want 4 (got text: %q)", mainCount, mainText)
	}
	if mainCount > allCount {
		t.Fatalf("main content count %d should never exceed total visible count %d", mainCount, allCount)
	}
}
