// Package issues provides SEO issue detection for crawled pages.
package issues

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/parser"
)

// DetectedIssue represents a single SEO issue found on a page.
type DetectedIssue struct {
	IssueType   string `json:"issueType"`
	Severity    string `json:"severity"`    // error, warning, info
	Scope       string `json:"scope"`       // always "page_local" for Phase 1
	DetailsJSON string `json:"detailsJson"` // JSON with issue-specific details
}

// PageContext holds all parsed and fetched data needed for issue detection.
type PageContext struct {
	// From fetch
	StatusCode           int
	RedirectHopCount     int
	RedirectLoopDetected bool
	RedirectHopsExceeded bool
	TTFBMS               int64
	ContentType          string

	// From parser
	Title                string
	TitleLength          int
	MetaDescription      string
	DescriptionLength    int
	MetaRobots           string
	XRobotsTag           string
	CanonicalType        string // self, cross, absent
	H1Count              int
	OGTitle              string
	OGDescription        string
	OGImage              string
	JSONLDBlocks         int
	MalformedJSONLD      bool
	JSONLDRaw            string
	WordCount            int
	MainContentWordCount int
	ImagesWithoutAlt     int
	ImagesWithEmptyAlt   int
	MixedContent           bool
	JSSuspect              bool
	ScriptCount            int
	HasSPARoot             bool
	TitleOutsideHead       bool
	MetaRobotsOutsideHead  bool

	// Batch A: title/meta, headings, canonicals
	H1s                        []string // all H1 texts
	H2s                        []string // all H2 texts
	TitleCount                 int
	DescriptionCount           int
	MetaDescriptionOutsideHead bool
	FirstHeadingLevel          int      // level of first heading (1-6), 0 if none
	H1AltTextOnly              []string // alt texts from H1s that contain only an <img>
	CanonicalCount             int
	CanonicalRaw               string
	CanonicalOutsideHead       bool

	// Image details (Batch B)
	Images []parser.DiscoveredImage

	// Edge details (Batch B) — populated from edges built during parse
	InternalOutlinkCount         int
	NonDescriptiveAnchorCount    int
	NonDescriptiveAnchorExamples []string
	InternalNofollowCount        int

	// URL of the page being analyzed (Batch B)
	PageURL string
}

// Thresholds holds configurable limits for issue detection.
type Thresholds struct {
	TitleMaxLength       int
	TitleMinLength       int
	DescriptionMaxLength int
	DescriptionMinLength int
	ThinContentThreshold int
	DeepPageThreshold    int
}

// DefaultThresholds returns sensible defaults for SEO issue detection.
func DefaultThresholds() Thresholds {
	return Thresholds{
		TitleMaxLength:       60,
		TitleMinLength:       30,
		DescriptionMaxLength: 160,
		DescriptionMinLength: 70,
		ThinContentThreshold: 200,
		DeepPageThreshold:    3,
	}
}

// DetectPageLocalIssues runs all Phase 1 detectors and returns found issues.
func DetectPageLocalIssues(ctx PageContext, thresholds Thresholds, depth int) []DetectedIssue {
	issues := []DetectedIssue{}

	// HTTP Status
	if ctx.StatusCode >= 400 && ctx.StatusCode <= 499 {
		issues = append(issues, newIssue("status_4xx", "error", map[string]any{
			"statusCode": ctx.StatusCode,
		}))
	}
	if ctx.StatusCode >= 500 && ctx.StatusCode <= 599 {
		issues = append(issues, newIssue("status_5xx", "error", map[string]any{
			"statusCode": ctx.StatusCode,
		}))
	}
	if ctx.RedirectHopCount > 1 {
		issues = append(issues, newIssue("redirect_chain", "warning", map[string]any{
			"hopCount": ctx.RedirectHopCount,
		}))
	}

	// Redirect
	if ctx.RedirectLoopDetected {
		issues = append(issues, newIssue("redirect_loop", "error", map[string]any{
			"detected": true,
		}))
	}
	if ctx.RedirectHopsExceeded {
		issues = append(issues, newIssue("redirect_hops_exceeded", "error", map[string]any{
			"hopCount": ctx.RedirectHopCount,
		}))
	}

	// Title
	if ctx.Title == "" {
		issues = append(issues, newIssue("missing_title", "error", map[string]any{}))
	}
	if ctx.TitleLength > thresholds.TitleMaxLength {
		issues = append(issues, newIssue("title_too_long", "warning", map[string]any{
			"length":    ctx.TitleLength,
			"maxLength": thresholds.TitleMaxLength,
			"title":     ctx.Title,
		}))
	}
	if ctx.TitleLength > 0 && ctx.TitleLength < thresholds.TitleMinLength {
		issues = append(issues, newIssue("title_too_short", "warning", map[string]any{
			"length":    ctx.TitleLength,
			"minLength": thresholds.TitleMinLength,
			"title":     ctx.Title,
		}))
	}

	// Description
	if ctx.MetaDescription == "" {
		issues = append(issues, newIssue("missing_description", "error", map[string]any{}))
	}
	if ctx.DescriptionLength > thresholds.DescriptionMaxLength {
		issues = append(issues, newIssue("description_too_long", "warning", map[string]any{
			"length":    ctx.DescriptionLength,
			"maxLength": thresholds.DescriptionMaxLength,
		}))
	}
	if ctx.DescriptionLength > 0 && ctx.DescriptionLength < thresholds.DescriptionMinLength {
		issues = append(issues, newIssue("description_too_short", "warning", map[string]any{
			"length":    ctx.DescriptionLength,
			"minLength": thresholds.DescriptionMinLength,
		}))
	}

	// Canonical
	if ctx.CanonicalType == "absent" {
		issues = append(issues, newIssue("missing_canonical", "warning", map[string]any{}))
	}

	// Headings
	if ctx.H1Count == 0 {
		issues = append(issues, newIssue("missing_h1", "warning", map[string]any{}))
	}
	if ctx.H1Count > 1 {
		issues = append(issues, newIssue("multiple_h1", "warning", map[string]any{
			"count": ctx.H1Count,
		}))
	}

	// Open Graph
	if ctx.OGTitle == "" {
		issues = append(issues, newIssue("missing_og_title", "info", map[string]any{}))
	}
	if ctx.OGDescription == "" {
		issues = append(issues, newIssue("missing_og_description", "info", map[string]any{}))
	}
	if ctx.OGImage == "" {
		issues = append(issues, newIssue("missing_og_image", "info", map[string]any{}))
	}

	// Structured Data
	if ctx.JSONLDBlocks == 0 {
		issues = append(issues, newIssue("missing_structured_data", "info", map[string]any{}))
	}
	if ctx.MalformedJSONLD {
		issues = append(issues, newIssue("malformed_structured_data", "warning", map[string]any{}))
	}

	// Validate structured data semantics
	if ctx.JSONLDRaw != "" && ctx.JSONLDRaw != "[]" {
		validationResults := parser.ValidateJSONLD(ctx.JSONLDRaw)
		for _, r := range validationResults {
			if len(r.MissingRequired) > 0 {
				issues = append(issues, newIssue("invalid_structured_data", "warning", map[string]any{
					"type":            r.Type,
					"missingRequired": r.MissingRequired,
					"scope":           "required",
				}))
			}
			if len(r.MissingRecommended) > 0 && !r.Nested {
				issues = append(issues, newIssue("incomplete_structured_data", "info", map[string]any{
					"type":               r.Type,
					"missingRecommended": r.MissingRecommended,
					"scope":              "recommended",
				}))
			}
		}
	}

	// Content
	if ctx.WordCount < thresholds.ThinContentThreshold {
		issues = append(issues, newIssue("thin_content", "warning", map[string]any{
			"wordCount": ctx.WordCount,
			"threshold": thresholds.ThinContentThreshold,
		}))
	}

	// Images
	if ctx.ImagesWithoutAlt > 0 {
		issues = append(issues, newIssue("missing_alt_attribute", "warning", map[string]any{
			"count": ctx.ImagesWithoutAlt,
		}))
	}
	if ctx.ImagesWithEmptyAlt > 0 {
		issues = append(issues, newIssue("empty_alt_attribute", "info", map[string]any{
			"count": ctx.ImagesWithEmptyAlt,
		}))
	}

	// Security
	if ctx.MixedContent {
		issues = append(issues, newIssue("mixed_content", "warning", map[string]any{}))
	}

	// Performance
	if ctx.TTFBMS > 10000 {
		issues = append(issues, newIssue("very_slow_response", "warning", map[string]any{
			"ttfbMs": ctx.TTFBMS,
		}))
	}
	if ctx.TTFBMS > 3000 {
		issues = append(issues, newIssue("slow_response", "info", map[string]any{
			"ttfbMs": ctx.TTFBMS,
		}))
	}

	// Depth
	if depth > thresholds.DeepPageThreshold {
		issues = append(issues, newIssue("deep_page", "info", map[string]any{
			"depth":     depth,
			"threshold": thresholds.DeepPageThreshold,
		}))
	}

	// Robots meta vs header mismatch
	if ctx.MetaRobots != "" && ctx.XRobotsTag != "" {
		metaDirectives := parseRobotsDirectives(ctx.MetaRobots)
		headerDirectives := parseRobotsDirectives(ctx.XRobotsTag)
		if !directivesMatch(metaDirectives, headerDirectives) {
			issues = append(issues, newIssue("robots_meta_header_mismatch", "warning", map[string]any{
				"metaRobots": ctx.MetaRobots,
				"xRobotsTag": ctx.XRobotsTag,
			}))
		}
	}

	// JS Rendering
	if ctx.JSSuspect {
		issues = append(issues, newIssue("js_suspect_not_rendered", "info", map[string]any{}))
	}

	// Tags outside <head>
	if ctx.TitleOutsideHead {
		issues = append(issues, newIssue("title_outside_head", "warning", map[string]any{}))
	}
	if ctx.MetaRobotsOutsideHead {
		issues = append(issues, newIssue("meta_robots_outside_head", "warning", map[string]any{}))
	}

	// ── Batch A: Title/Meta + Headings + Canonicals ────────────────────

	// title_same_as_h1
	if ctx.Title != "" && len(ctx.H1s) > 0 {
		if strings.EqualFold(strings.TrimSpace(ctx.Title), strings.TrimSpace(ctx.H1s[0])) {
			issues = append(issues, newIssue("title_same_as_h1", "warning", map[string]any{
				"title": ctx.Title,
				"h1":    ctx.H1s[0],
			}))
		}
	}

	// multiple_title_tags
	if ctx.TitleCount > 1 {
		issues = append(issues, newIssue("multiple_title_tags", "warning", map[string]any{
			"count": ctx.TitleCount,
		}))
	}

	// multiple_meta_descriptions
	if ctx.DescriptionCount > 1 {
		issues = append(issues, newIssue("multiple_meta_descriptions", "warning", map[string]any{
			"count": ctx.DescriptionCount,
		}))
	}

	// meta_description_outside_head
	if ctx.MetaDescriptionOutsideHead {
		issues = append(issues, newIssue("meta_description_outside_head", "warning", map[string]any{}))
	}

	// h1_too_long
	for _, h1 := range ctx.H1s {
		if len([]rune(h1)) > 70 {
			issues = append(issues, newIssue("h1_too_long", "info", map[string]any{
				"length": len([]rune(h1)),
				"h1":     h1,
			}))
		}
	}

	// h1_non_sequential
	if ctx.FirstHeadingLevel > 1 {
		issues = append(issues, newIssue("h1_non_sequential", "warning", map[string]any{
			"firstHeadingLevel": ctx.FirstHeadingLevel,
		}))
	}

	// h1_alt_text_only
	for _, alt := range ctx.H1AltTextOnly {
		issues = append(issues, newIssue("h1_alt_text_only", "warning", map[string]any{
			"alt": alt,
		}))
	}

	// missing_h2
	if len(ctx.H2s) == 0 {
		issues = append(issues, newIssue("missing_h2", "info", map[string]any{}))
	}

	// h2_non_sequential: H2 without a preceding H1
	if len(ctx.H2s) > 0 && ctx.H1Count == 0 {
		issues = append(issues, newIssue("h2_non_sequential", "warning", map[string]any{}))
	} else if len(ctx.H2s) > 0 && ctx.FirstHeadingLevel > 1 {
		// First heading is H2+ meaning H2 appears before H1
		issues = append(issues, newIssue("h2_non_sequential", "warning", map[string]any{}))
	}

	// h2_too_long
	for _, h2 := range ctx.H2s {
		if len([]rune(h2)) > 70 {
			issues = append(issues, newIssue("h2_too_long", "info", map[string]any{
				"length": len([]rune(h2)),
				"h2":     h2,
			}))
		}
	}

	// multiple_canonicals
	if ctx.CanonicalCount > 1 {
		issues = append(issues, newIssue("multiple_canonicals", "warning", map[string]any{
			"count": ctx.CanonicalCount,
		}))
	}

	// canonical_is_relative
	if ctx.CanonicalRaw != "" && !strings.HasPrefix(strings.ToLower(ctx.CanonicalRaw), "http") {
		issues = append(issues, newIssue("canonical_is_relative", "warning", map[string]any{
			"canonical": ctx.CanonicalRaw,
		}))
	}

	// canonical_outside_head
	if ctx.CanonicalOutsideHead {
		issues = append(issues, newIssue("canonical_outside_head", "warning", map[string]any{}))
	}

	// ── Batch B: Image issues ──────────────────────────────────────────

	// alt_text_too_long
	if len(ctx.Images) > 0 {
		var longAltCount int
		var maxLen int
		for _, img := range ctx.Images {
			l := len(img.Alt)
			if l > 100 {
				longAltCount++
				if l > maxLen {
					maxLen = l
				}
			}
		}
		if longAltCount > 0 {
			issues = append(issues, newIssue("alt_text_too_long", "warning", map[string]any{
				"count":     longAltCount,
				"maxLength": maxLen,
			}))
		}
	}

	// missing_image_size_attributes
	if len(ctx.Images) > 0 {
		var missingSizeCount int
		for _, img := range ctx.Images {
			if !img.HasWidth && !img.HasHeight {
				missingSizeCount++
			}
		}
		if missingSizeCount > 0 {
			issues = append(issues, newIssue("missing_image_size_attributes", "info", map[string]any{
				"count": missingSizeCount,
			}))
		}
	}

	// ── Batch B: Link issues ───────────────────────────────────────────

	// non_descriptive_anchor_text
	if ctx.NonDescriptiveAnchorCount > 0 {
		issues = append(issues, newIssue("non_descriptive_anchor_text", "warning", map[string]any{
			"count":    ctx.NonDescriptiveAnchorCount,
			"examples": ctx.NonDescriptiveAnchorExamples,
		}))
	}

	// internal_nofollow_outlink
	if ctx.InternalNofollowCount > 0 {
		issues = append(issues, newIssue("internal_nofollow_outlink", "warning", map[string]any{
			"count": ctx.InternalNofollowCount,
		}))
	}

	// ── Batch B: URL issues ────────────────────────────────────────────
	issues = append(issues, DetectURLIssues(ctx.PageURL)...)

	return issues
}

// nonDescriptiveAnchors is the set of generic anchor texts considered non-descriptive.
var nonDescriptiveAnchors = map[string]bool{
	"click here": true,
	"read more":  true,
	"learn more": true,
	"here":       true,
	"this":       true,
	"more":       true,
	"link":       true,
	"go":         true,
	"visit":      true,
}

// IsNonDescriptiveAnchor checks if the given anchor text is generic/non-descriptive.
func IsNonDescriptiveAnchor(anchor string) bool {
	return nonDescriptiveAnchors[strings.ToLower(strings.TrimSpace(anchor))]
}

// DetectURLIssues checks a URL string for common SEO URL issues.
func DetectURLIssues(rawURL string) []DetectedIssue {
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}

	var issues []DetectedIssue

	// url_uppercase — path contains uppercase letters
	if hasUppercase(parsed.Path) {
		issues = append(issues, newIssue("url_uppercase", "info", map[string]any{
			"url": rawURL,
		}))
	}

	// url_underscores — path contains underscores
	if strings.Contains(parsed.Path, "_") {
		issues = append(issues, newIssue("url_underscores", "info", map[string]any{
			"url": rawURL,
		}))
	}

	// url_contains_space — encoded spaces (%20 or +)
	if strings.Contains(rawURL, "%20") || strings.Contains(parsed.RawQuery, "+") || strings.Contains(parsed.Path, "+") {
		issues = append(issues, newIssue("url_contains_space", "warning", map[string]any{
			"url": rawURL,
		}))
	}

	// url_has_parameters — has query string
	if parsed.RawQuery != "" {
		params := []string{}
		for key := range parsed.Query() {
			params = append(params, key)
		}
		issues = append(issues, newIssue("url_has_parameters", "info", map[string]any{
			"url":    rawURL,
			"params": params,
		}))
	}

	// url_too_long — over 115 characters
	if len(rawURL) > 115 {
		issues = append(issues, newIssue("url_too_long", "info", map[string]any{
			"url":    rawURL,
			"length": len(rawURL),
		}))
	}

	// url_multiple_slashes — consecutive slashes in path
	if strings.Contains(parsed.Path, "//") {
		issues = append(issues, newIssue("url_multiple_slashes", "warning", map[string]any{
			"url": rawURL,
		}))
	}

	// url_repetitive_path — repeating path segments
	if seg := findRepetitiveSegment(parsed.Path); seg != "" {
		issues = append(issues, newIssue("url_repetitive_path", "warning", map[string]any{
			"url":             rawURL,
			"repeatedSegment": seg,
		}))
	}

	return issues
}

// hasUppercase returns true if the string contains any uppercase letter.
func hasUppercase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// findRepetitiveSegment returns the first repeated consecutive path segment, or "".
func findRepetitiveSegment(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) < 2 {
		return ""
	}
	for i := 1; i < len(segments); i++ {
		if segments[i] != "" && segments[i] == segments[i-1] {
			return segments[i]
		}
	}
	return ""
}

// parseRobotsDirectives splits a robots directive string into a normalized set.
func parseRobotsDirectives(raw string) map[string]bool {
	directives := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		d := strings.TrimSpace(strings.ToLower(part))
		if d != "" {
			directives[d] = true
		}
	}
	return directives
}

// directivesMatch returns true if two directive sets are identical.
func directivesMatch(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func newIssue(issueType, severity string, details map[string]any) DetectedIssue {
	detailsBytes, err := json.Marshal(details)
	if err != nil {
		detailsBytes = []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
	}
	return DetectedIssue{
		IssueType:   issueType,
		Severity:    severity,
		Scope:       "page_local",
		DetailsJSON: string(detailsBytes),
	}
}
