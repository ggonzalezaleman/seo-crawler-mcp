// Package issues provides SEO issue detection for crawled pages.
package issues

import (
	"encoding/json"
	"fmt"
	"strings"

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
	MixedContent         bool
	JSSuspect            bool
	ScriptCount          int
	HasSPARoot           bool
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

	return issues
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
