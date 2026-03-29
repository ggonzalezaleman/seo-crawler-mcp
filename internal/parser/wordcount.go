package parser

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func extractVisibleTextFromSelection(sel *goquery.Selection) string {
	var buf strings.Builder
	var extract func(*goquery.Selection)
	extract = func(s *goquery.Selection) {
		s.Contents().Each(func(_ int, child *goquery.Selection) {
			if child.Is("script, style, noscript") {
				return
			}
			if goquery.NodeName(child) == "#text" {
				buf.WriteString(child.Text())
				buf.WriteByte(' ')
			} else {
				extract(child)
			}
		})
	}
	extract(sel)
	return strings.TrimSpace(buf.String())
}

// ExtractVisibleText extracts all visible text from an HTML document
// without mutating the original DOM tree.
func ExtractVisibleText(doc *goquery.Document) string {
	return extractVisibleTextFromSelection(doc.Find("body"))
}

// ExtractMainContentText extracts text from main content areas.
// Priority: <main>, <article>, [role="main"].
// Fallback: <body> excluding <nav>, <header>, <footer>, <aside>.
func ExtractMainContentText(doc *goquery.Document) string {
	// Try primary content selectors.
	mainSel := doc.Find("main, article, [role='main']")
	if mainSel.Length() > 0 {
		var parts []string
		mainSel.Each(func(_ int, s *goquery.Selection) {
			text := extractVisibleTextFromSelection(s)
			if text != "" {
				parts = append(parts, text)
			}
		})
		return strings.Join(parts, " ")
	}

	// Fallback: body minus nav/header/footer/aside, using the same visible-text
	// extraction path as total word count so main content can never exceed it due
	// to script/style/noscript or hidden-like artifacts from Selection.Text().
	body := doc.Find("body").Clone()
	body.Find("nav, header, footer, aside").Remove()
	return extractVisibleTextFromSelection(body)
}

// CountWords counts non-empty whitespace-separated tokens.
func CountWords(text string) int {
	fields := strings.Fields(text)
	return len(fields)
}
