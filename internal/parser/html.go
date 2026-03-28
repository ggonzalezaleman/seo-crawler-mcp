// Package parser extracts SEO metadata from HTML documents.
package parser

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/encoding"
)

// RelLink represents a rel=next or rel=prev link.
type RelLink struct {
	Raw      string `json:"raw"`
	Resolved string `json:"resolved"`
}

// HreflangEntry represents a single hreflang alternate link.
type HreflangEntry struct {
	Lang string `json:"lang"`
	URL  string `json:"url"`
}

// HeadingSet holds headings by level.
type HeadingSet struct {
	H1 []string `json:"h1"`
	H2 []string `json:"h2"`
	H3 []string `json:"h3"`
	H4 []string `json:"h4"`
	H5 []string `json:"h5"`
	H6 []string `json:"h6"`
}

// OGTags holds Open Graph metadata.
type OGTags struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	Type        string `json:"type,omitempty"`
	URL         string `json:"url,omitempty"`
}

// TwitterTags holds Twitter Card metadata.
type TwitterTags struct {
	Card        string `json:"card,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
}

// JSONLDBlock holds a single JSON-LD script block.
type JSONLDBlock struct {
	Raw       string `json:"raw"`
	Type      string `json:"type,omitempty"`
	Malformed bool   `json:"malformed,omitempty"`
}

// DiscoveredLink represents a link found in the page.
type DiscoveredLink struct {
	URL        string `json:"url"`
	AnchorText string `json:"anchorText"`
	Rel        string `json:"rel,omitempty"`
}

// DiscoveredImage represents an image found in the page.
type DiscoveredImage struct {
	Src       string `json:"src"`
	Alt       string `json:"alt"`
	AltEmpty  bool   `json:"altEmpty"`
	AltMissing bool  `json:"altMissing"`
}

// ParseResult holds all extracted SEO data from an HTML page.
type ParseResult struct {
	Title                string
	TitleLength          int
	MetaDescription      string
	DescriptionLength    int
	MetaRobots           string
	XRobotsTag           string
	IndexabilityState    string
	CanonicalRaw         string
	CanonicalResolved    string
	CanonicalType        string // self, cross, absent
	RelNext              *RelLink
	RelPrev              *RelLink
	Hreflangs            []HreflangEntry
	Headings             HeadingSet
	OpenGraph            OGTags
	TwitterCard          TwitterTags
	JSONLDBlocks         []JSONLDBlock
	JSONLDTypes          []string
	Links                []DiscoveredLink
	Images               []DiscoveredImage
	ExtractedWordCount   int
	MainContentWordCount int
	ContentHash          string
	JSSuspect            bool
	ScriptCount          int
	HasSPARoot           bool
}

// ParseHTML extracts SEO metadata from raw HTML bytes.
func ParseHTML(body []byte, pageURL string, responseHeaders http.Header) (*ParseResult, error) {
	utf8Body, err := encoding.DetectAndConvert(body, responseHeaders.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("encoding conversion: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(utf8Body))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	baseURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("parsing page URL %q: %w", pageURL, err)
	}

	r := &ParseResult{
		Hreflangs:  make([]HreflangEntry, 0),
		JSONLDBlocks: make([]JSONLDBlock, 0),
		JSONLDTypes: make([]string, 0),
		Links:      make([]DiscoveredLink, 0),
		Images:     make([]DiscoveredImage, 0),
	}
	r.Headings = HeadingSet{
		H1: make([]string, 0),
		H2: make([]string, 0),
		H3: make([]string, 0),
		H4: make([]string, 0),
		H5: make([]string, 0),
		H6: make([]string, 0),
	}

	// Title
	r.Title = strings.TrimSpace(doc.Find("title").First().Text())
	r.TitleLength = len(r.Title)

	// Meta description
	r.MetaDescription = doc.Find(`meta[name="description"]`).AttrOr("content", "")
	r.DescriptionLength = len(r.MetaDescription)

	// Meta robots
	r.MetaRobots = doc.Find(`meta[name="robots"]`).AttrOr("content", "")

	// X-Robots-Tag
	r.XRobotsTag = responseHeaders.Get("X-Robots-Tag")

	// Indexability
	r.IndexabilityState = "indexable"
	if containsDirective(r.MetaRobots, "noindex") {
		r.IndexabilityState = "noindex_meta"
	} else if containsDirective(r.XRobotsTag, "noindex") {
		r.IndexabilityState = "noindex_header"
	}

	// Canonical
	extractCanonical(doc, baseURL, pageURL, r)

	// Rel next/prev
	doc.Find(`link[rel="next"]`).Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			resolved := resolveURL(baseURL, href)
			r.RelNext = &RelLink{Raw: href, Resolved: resolved}
		}
	})
	doc.Find(`link[rel="prev"]`).Each(func(_ int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			resolved := resolveURL(baseURL, href)
			r.RelPrev = &RelLink{Raw: href, Resolved: resolved}
		}
	})

	// Hreflang
	doc.Find(`link[rel="alternate"][hreflang]`).Each(func(_ int, s *goquery.Selection) {
		lang, _ := s.Attr("hreflang")
		href, _ := s.Attr("href")
		if lang != "" && href != "" {
			r.Hreflangs = append(r.Hreflangs, HreflangEntry{
				Lang: lang,
				URL:  resolveURL(baseURL, href),
			})
		}
	})

	// Headings
	for level := 1; level <= 6; level++ {
		tag := fmt.Sprintf("h%d", level)
		doc.Find(tag).Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text == "" {
				return
			}
			switch level {
			case 1:
				r.Headings.H1 = append(r.Headings.H1, text)
			case 2:
				r.Headings.H2 = append(r.Headings.H2, text)
			case 3:
				r.Headings.H3 = append(r.Headings.H3, text)
			case 4:
				r.Headings.H4 = append(r.Headings.H4, text)
			case 5:
				r.Headings.H5 = append(r.Headings.H5, text)
			case 6:
				r.Headings.H6 = append(r.Headings.H6, text)
			}
		})
	}

	// OG tags
	r.OpenGraph = OGTags{
		Title:       metaProperty(doc, "og:title"),
		Description: metaProperty(doc, "og:description"),
		Image:       metaProperty(doc, "og:image"),
		Type:        metaProperty(doc, "og:type"),
		URL:         metaProperty(doc, "og:url"),
	}

	// Twitter cards
	r.TwitterCard = TwitterTags{
		Card:        metaName(doc, "twitter:card"),
		Title:       metaName(doc, "twitter:title"),
		Description: metaName(doc, "twitter:description"),
		Image:       metaName(doc, "twitter:image"),
	}

	// JSON-LD
	doc.Find(`script[type="application/ld+json"]`).Each(func(_ int, s *goquery.Selection) {
		raw := strings.TrimSpace(s.Text())
		block := JSONLDBlock{Raw: raw}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			block.Malformed = true
		} else if t, ok := parsed["@type"]; ok {
			if ts, ok := t.(string); ok {
				block.Type = ts
				r.JSONLDTypes = append(r.JSONLDTypes, ts)
			}
		}
		r.JSONLDBlocks = append(r.JSONLDBlocks, block)
	})

	// Links
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		href = strings.TrimSpace(href)
		if shouldSkipHref(href) {
			return
		}
		resolved := resolveURL(baseURL, href)
		rel, _ := s.Attr("rel")
		r.Links = append(r.Links, DiscoveredLink{
			URL:        resolved,
			AnchorText: strings.TrimSpace(s.Text()),
			Rel:        rel,
		})
	})

	// Images
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		src := s.AttrOr("src", "")
		alt, altExists := s.Attr("alt")
		img := DiscoveredImage{
			Src:        resolveURL(baseURL, src),
			Alt:        alt,
			AltEmpty:   altExists && alt == "",
			AltMissing: !altExists,
		}
		r.Images = append(r.Images, img)
	})

	// Script count + SPA root detection
	r.ScriptCount = doc.Find("script").Length()
	spaIDs := []string{"root", "__next", "app", "__nuxt"}
	for _, id := range spaIDs {
		if doc.Find("#" + id).Length() > 0 {
			r.HasSPARoot = true
			break
		}
	}

	// Word counts
	allText := ExtractVisibleText(doc)
	r.ExtractedWordCount = CountWords(allText)

	mainText := ExtractMainContentText(doc)
	r.MainContentWordCount = CountWords(mainText)

	// Content hash
	hash := sha256.Sum256([]byte(allText))
	r.ContentHash = fmt.Sprintf("%x", hash)

	// JS suspect
	r.JSSuspect = r.ExtractedWordCount < 50 && (r.HasSPARoot || r.ScriptCount >= 5)

	return r, nil
}

func extractCanonical(doc *goquery.Document, baseURL *url.URL, pageURL string, r *ParseResult) {
	canonical := doc.Find(`link[rel="canonical"]`).AttrOr("href", "")
	if canonical == "" {
		r.CanonicalType = "absent"
		return
	}

	r.CanonicalRaw = canonical
	r.CanonicalResolved = resolveURL(baseURL, canonical)

	if normalizeForComparison(r.CanonicalResolved) == normalizeForComparison(pageURL) {
		r.CanonicalType = "self"
	} else {
		r.CanonicalType = "cross"
	}
}

func resolveURL(base *url.URL, raw string) string {
	ref, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return base.ResolveReference(ref).String()
}

func normalizeForComparison(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	// Remove trailing slash for comparison.
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

func containsDirective(val, directive string) bool {
	for _, part := range strings.Split(val, ",") {
		if strings.TrimSpace(strings.ToLower(part)) == directive {
			return true
		}
	}
	return false
}

func metaProperty(doc *goquery.Document, property string) string {
	return doc.Find(fmt.Sprintf(`meta[property="%s"]`, property)).AttrOr("content", "")
}

func metaName(doc *goquery.Document, name string) string {
	return doc.Find(fmt.Sprintf(`meta[name="%s"]`, name)).AttrOr("content", "")
}

var skipPrefixes = []string{"javascript:", "mailto:", "tel:", "data:", "blob:"}

func shouldSkipHref(href string) bool {
	lower := strings.ToLower(strings.TrimSpace(href))
	if lower == "" || lower == "#" {
		return true
	}
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}
