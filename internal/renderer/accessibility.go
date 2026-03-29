package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"time"
)

// AxeResult holds the results from an axe-core accessibility audit.
type AxeResult struct {
	URL        string     `json:"url"`
	Violations []AxeIssue `json:"violations"`
	Passes     int        `json:"passes"`
	Incomplete int        `json:"incomplete"`
}

// AxeIssue represents a single accessibility violation found by axe-core.
type AxeIssue struct {
	ID          string   `json:"id"`
	Impact      string   `json:"impact"` // "critical", "serious", "moderate", "minor"
	Description string   `json:"description"`
	Help        string   `json:"help"`
	HelpURL     string   `json:"helpUrl"`
	Tags        []string `json:"tags"` // WCAG tags
	Nodes       int      `json:"nodes"` // Number of affected elements
}

// IsPublicURL returns true if the URL points to a public host (not localhost/loopback).
func IsPublicURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return host != "localhost" && host != "127.0.0.1" && host != "::1" && host != "0.0.0.0"
}

// RunAxeAudit runs an axe-core accessibility audit on a URL using Playwright.
// Requires python3 and playwright to be installed. Skips non-public URLs.
func RunAxeAudit(ctx context.Context, pageURL string) (*AxeResult, error) {
	if !IsPublicURL(pageURL) {
		return nil, fmt.Errorf("skipping axe audit for non-public URL: %s", pageURL)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-c", axeScript(), pageURL)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("axe audit failed: %w", err)
	}

	var result AxeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parsing axe output: %w", err)
	}
	return &result, nil
}

func axeScript() string {
	return `
import sys, json
from playwright.sync_api import sync_playwright

url = sys.argv[1]

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    page = browser.new_page(viewport={"width": 1440, "height": 900})
    page.goto(url, wait_until="networkidle", timeout=30000)
    page.wait_for_timeout(2000)

    # Inject axe-core from CDN
    page.add_script_tag(url="https://cdnjs.cloudflare.com/ajax/libs/axe-core/4.10.2/axe.min.js")
    page.wait_for_timeout(1000)

    # Run axe
    results = page.evaluate("""
        () => new Promise((resolve, reject) => {
            axe.run(document, {
                runOnly: {
                    type: 'tag',
                    values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'best-practice']
                }
            }).then(results => {
                resolve({
                    violations: results.violations.map(v => ({
                        id: v.id,
                        impact: v.impact,
                        description: v.description,
                        help: v.help,
                        helpUrl: v.helpUrl,
                        tags: v.tags,
                        nodes: v.nodes.length
                    })),
                    passes: results.passes.length,
                    incomplete: results.incomplete.length
                });
            }).catch(reject);
        })
    """)

    result = {
        "url": url,
        "violations": results["violations"],
        "passes": results["passes"],
        "incomplete": results["incomplete"]
    }

    browser.close()

print(json.dumps(result))
`
}
