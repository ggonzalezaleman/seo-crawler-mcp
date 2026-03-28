package issues

import (
	"strings"
	"testing"
)

// cleanPage returns a well-formed PageContext that should produce no error/warning issues.
func cleanPage() PageContext {
	return PageContext{
		StatusCode:       200,
		RedirectHopCount: 0,
		TTFBMS:           500,
		ContentType:      "text/html",
		Title:            "A Good SEO Title That Is Just Right",
		TitleLength:      36,
		MetaDescription:  "This is a well-crafted meta description that provides enough detail for search engines.",
		DescriptionLength: 88,
		CanonicalType:    "self",
		H1Count:          1,
		OGTitle:          "OG Title",
		OGDescription:    "OG Description",
		OGImage:          "https://example.com/og.png",
		JSONLDBlocks:     1,
		WordCount:        500,
		MainContentWordCount: 400,
	}
}

func defaultThresholds() Thresholds {
	return DefaultThresholds()
}

func hasIssue(issues []DetectedIssue, issueType string) bool {
	for _, issue := range issues {
		if issue.IssueType == issueType {
			return true
		}
	}
	return false
}

func countIssues(issues []DetectedIssue, issueType string) int {
	count := 0
	for _, issue := range issues {
		if issue.IssueType == issueType {
			count++
		}
	}
	return count
}

func TestCleanPage_NoIssues(t *testing.T) {
	issues := DetectPageLocalIssues(cleanPage(), defaultThresholds(), 1)
	for _, issue := range issues {
		if issue.Severity == "error" || issue.Severity == "warning" {
			t.Errorf("clean page should have no error/warning issues, got %q (%s)", issue.IssueType, issue.Severity)
		}
	}
}

func TestDetectPageLocalIssues(t *testing.T) {
	tests := []struct {
		name       string
		ctx        PageContext
		thresholds Thresholds
		depth      int
		wantTypes  []string
		wantAbsent []string
	}{
		{
			name:       "clean page — zero issues",
			ctx:        cleanPage(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{},
		},
		{
			name: "404 page",
			ctx: func() PageContext {
				p := cleanPage()
				p.StatusCode = 404
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"status_4xx"},
		},
		{
			name: "500 page",
			ctx: func() PageContext {
				p := cleanPage()
				p.StatusCode = 503
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"status_5xx"},
		},
		{
			name: "redirect chain (3 hops)",
			ctx: func() PageContext {
				p := cleanPage()
				p.RedirectHopCount = 3
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"redirect_chain"},
		},
		{
			name: "redirect loop",
			ctx: func() PageContext {
				p := cleanPage()
				p.RedirectLoopDetected = true
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"redirect_loop"},
		},
		{
			name: "redirect hops exceeded",
			ctx: func() PageContext {
				p := cleanPage()
				p.RedirectHopsExceeded = true
				p.RedirectHopCount = 10
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"redirect_hops_exceeded"},
		},
		{
			name: "missing title",
			ctx: func() PageContext {
				p := cleanPage()
				p.Title = ""
				p.TitleLength = 0
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_title"},
		},
		{
			name: "title too long (75 chars)",
			ctx: func() PageContext {
				p := cleanPage()
				p.Title = strings.Repeat("A", 75)
				p.TitleLength = 75
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"title_too_long"},
		},
		{
			name: "title too short (10 chars)",
			ctx: func() PageContext {
				p := cleanPage()
				p.Title = "Short"
				p.TitleLength = 10
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"title_too_short"},
		},
		{
			name: "missing description",
			ctx: func() PageContext {
				p := cleanPage()
				p.MetaDescription = ""
				p.DescriptionLength = 0
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_description"},
		},
		{
			name: "description too long",
			ctx: func() PageContext {
				p := cleanPage()
				p.MetaDescription = strings.Repeat("D", 200)
				p.DescriptionLength = 200
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"description_too_long"},
		},
		{
			name: "description too short",
			ctx: func() PageContext {
				p := cleanPage()
				p.MetaDescription = "Short desc"
				p.DescriptionLength = 30
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"description_too_short"},
		},
		{
			name: "missing canonical",
			ctx: func() PageContext {
				p := cleanPage()
				p.CanonicalType = "absent"
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_canonical"},
		},
		{
			name: "no H1",
			ctx: func() PageContext {
				p := cleanPage()
				p.H1Count = 0
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_h1"},
		},
		{
			name: "multiple H1 (3)",
			ctx: func() PageContext {
				p := cleanPage()
				p.H1Count = 3
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"multiple_h1"},
		},
		{
			name: "no OG tags",
			ctx: func() PageContext {
				p := cleanPage()
				p.OGTitle = ""
				p.OGDescription = ""
				p.OGImage = ""
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_og_title", "missing_og_description", "missing_og_image"},
		},
		{
			name: "no structured data",
			ctx: func() PageContext {
				p := cleanPage()
				p.JSONLDBlocks = 0
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_structured_data"},
		},
		{
			name: "malformed JSON-LD",
			ctx: func() PageContext {
				p := cleanPage()
				p.MalformedJSONLD = true
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"malformed_structured_data"},
		},
		{
			name: "thin content (45 words)",
			ctx: func() PageContext {
				p := cleanPage()
				p.WordCount = 45
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"thin_content"},
		},
		{
			name: "missing alt (3 images)",
			ctx: func() PageContext {
				p := cleanPage()
				p.ImagesWithoutAlt = 3
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_alt_attribute"},
		},
		{
			name: "empty alt attribute",
			ctx: func() PageContext {
				p := cleanPage()
				p.ImagesWithEmptyAlt = 2
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"empty_alt_attribute"},
		},
		{
			name: "mixed content",
			ctx: func() PageContext {
				p := cleanPage()
				p.MixedContent = true
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"mixed_content"},
		},
		{
			name: "slow response (4500ms)",
			ctx: func() PageContext {
				p := cleanPage()
				p.TTFBMS = 4500
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"slow_response"},
			wantAbsent: []string{"very_slow_response"},
		},
		{
			name: "very slow response (12000ms) — both slow and very_slow",
			ctx: func() PageContext {
				p := cleanPage()
				p.TTFBMS = 12000
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"very_slow_response", "slow_response"},
		},
		{
			name: "deep page (depth 5)",
			ctx:        cleanPage(),
			thresholds: defaultThresholds(),
			depth:      5,
			wantTypes:  []string{"deep_page"},
		},
		{
			name: "JS suspect",
			ctx: func() PageContext {
				p := cleanPage()
				p.JSSuspect = true
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"js_suspect_not_rendered"},
		},
		{
			name: "robots meta/header mismatch — noindex vs index",
			ctx: func() PageContext {
				p := cleanPage()
				p.MetaRobots = "noindex, follow"
				p.XRobotsTag = "index, follow"
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"robots_meta_header_mismatch"},
		},
		{
			name: "robots meta/header match — no issue",
			ctx: func() PageContext {
				p := cleanPage()
				p.MetaRobots = "noindex, nofollow"
				p.XRobotsTag = "nofollow, noindex"
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{},
			wantAbsent: []string{"robots_meta_header_mismatch"},
		},
		{
			name: "robots only meta — no mismatch",
			ctx: func() PageContext {
				p := cleanPage()
				p.MetaRobots = "noindex"
				p.XRobotsTag = ""
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantAbsent: []string{"robots_meta_header_mismatch"},
		},
		{
			name: "multiple issues — missing title + thin content + no H1",
			ctx: func() PageContext {
				p := cleanPage()
				p.Title = ""
				p.TitleLength = 0
				p.WordCount = 45
				p.H1Count = 0
				return p
			}(),
			thresholds: defaultThresholds(),
			depth:      1,
			wantTypes:  []string{"missing_title", "thin_content", "missing_h1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPageLocalIssues(tt.ctx, tt.thresholds, tt.depth)

			for _, wantType := range tt.wantTypes {
				if !hasIssue(got, wantType) {
					t.Errorf("expected issue %q not found in results", wantType)
				}
			}

			for _, absentType := range tt.wantAbsent {
				if hasIssue(got, absentType) {
					t.Errorf("unexpected issue %q found in results", absentType)
				}
			}

			// Verify all issues have scope "page_local"
			for _, issue := range got {
				if issue.Scope != "page_local" {
					t.Errorf("issue %q has scope %q, want %q", issue.IssueType, issue.Scope, "page_local")
				}
				if issue.DetailsJSON == "" {
					t.Errorf("issue %q has empty detailsJson", issue.IssueType)
				}
			}
		})
	}
}

func TestDetectedIssue_DetailsJSON(t *testing.T) {
	ctx := cleanPage()
	ctx.StatusCode = 404
	issues := DetectPageLocalIssues(ctx, defaultThresholds(), 1)

	for _, issue := range issues {
		if issue.IssueType == "status_4xx" {
			if !strings.Contains(issue.DetailsJSON, `"statusCode":404`) {
				t.Errorf("status_4xx detailsJson should contain statusCode, got %s", issue.DetailsJSON)
			}
			return
		}
	}
	t.Error("status_4xx issue not found")
}

func TestSliceNeverNil(t *testing.T) {
	// Even with zero issues matching, the returned slice must not be nil.
	issues := DetectPageLocalIssues(cleanPage(), defaultThresholds(), 1)
	if issues == nil {
		t.Error("returned slice must not be nil")
	}
}

func TestParseRobotsDirectives(t *testing.T) {
	tests := []struct {
		input string
		want  map[string]bool
	}{
		{"noindex, nofollow", map[string]bool{"noindex": true, "nofollow": true}},
		{"NOINDEX", map[string]bool{"noindex": true}},
		{"index, follow", map[string]bool{"index": true, "follow": true}},
		{"", map[string]bool{}},
	}
	for _, tt := range tests {
		got := parseRobotsDirectives(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("parseRobotsDirectives(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for k := range tt.want {
			if !got[k] {
				t.Errorf("parseRobotsDirectives(%q) missing %q", tt.input, k)
			}
		}
	}
}

func TestDirectivesMatch(t *testing.T) {
	a := map[string]bool{"noindex": true, "nofollow": true}
	b := map[string]bool{"nofollow": true, "noindex": true}
	c := map[string]bool{"index": true, "follow": true}

	if !directivesMatch(a, b) {
		t.Error("expected a and b to match")
	}
	if directivesMatch(a, c) {
		t.Error("expected a and c to not match")
	}
}

func TestIssueCounts(t *testing.T) {
	// Each detector should fire exactly once.
	ctx := cleanPage()
	ctx.StatusCode = 404
	issues := DetectPageLocalIssues(ctx, defaultThresholds(), 1)
	if count := countIssues(issues, "status_4xx"); count != 1 {
		t.Errorf("expected exactly 1 status_4xx issue, got %d", count)
	}
}

func TestDetectPageLocalIssues_InvalidStructuredData(t *testing.T) {
	ctx := cleanPage()
	// BlogPosting missing headline — should emit invalid_structured_data
	ctx.JSONLDRaw = `[{"raw":"{\"@type\":\"BlogPosting\",\"author\":\"Jane\",\"datePublished\":\"2024-01-01\"}","type":"BlogPosting"}]`
	issues := DetectPageLocalIssues(ctx, defaultThresholds(), 1)

	if !hasIssue(issues, "invalid_structured_data") {
		t.Error("expected invalid_structured_data issue for BlogPosting missing headline")
	}
}

func TestDetectPageLocalIssues_IncompleteStructuredData(t *testing.T) {
	ctx := cleanPage()
	// Organization with name but missing logo (recommended) — should emit incomplete_structured_data
	ctx.JSONLDRaw = `[{"raw":"{\"@type\":\"Organization\",\"name\":\"My Org\"}","type":"Organization"}]`
	issues := DetectPageLocalIssues(ctx, defaultThresholds(), 1)

	if !hasIssue(issues, "incomplete_structured_data") {
		t.Error("expected incomplete_structured_data issue for Organization missing recommended props")
	}
	// Should NOT have invalid_structured_data (name is present = all required met)
	if hasIssue(issues, "invalid_structured_data") {
		t.Error("unexpected invalid_structured_data — Organization has all required fields")
	}
}

func TestTitleOutsideHeadIssue(t *testing.T) {
	ctx := cleanPage()
	ctx.TitleOutsideHead = true
	issues := DetectPageLocalIssues(ctx, defaultThresholds(), 0)
	if !hasIssue(issues, "title_outside_head") {
		t.Error("expected title_outside_head issue")
	}
}

func TestMetaRobotsOutsideHeadIssue(t *testing.T) {
	ctx := cleanPage()
	ctx.MetaRobotsOutsideHead = true
	issues := DetectPageLocalIssues(ctx, defaultThresholds(), 0)
	if !hasIssue(issues, "meta_robots_outside_head") {
		t.Error("expected meta_robots_outside_head issue")
	}
}

func TestNoOutsideHeadIssuesOnCleanPage(t *testing.T) {
	ctx := cleanPage()
	issues := DetectPageLocalIssues(ctx, defaultThresholds(), 0)
	if hasIssue(issues, "title_outside_head") {
		t.Error("unexpected title_outside_head on clean page")
	}
	if hasIssue(issues, "meta_robots_outside_head") {
		t.Error("unexpected meta_robots_outside_head on clean page")
	}
}
