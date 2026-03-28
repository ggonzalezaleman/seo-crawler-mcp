package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// PlaywrightResult holds the output from the Playwright menu discovery script.
type PlaywrightResult struct {
	HTML  string   `json:"html"`
	Links []string `json:"links"`
}

// playwrightAvailable caches the result of the availability check.
var (
	playwrightOnce      sync.Once
	playwrightAvailable bool
)

// IsPlaywrightAvailable returns true if python3 and the playwright package
// are installed. The result is cached after the first call.
func IsPlaywrightAvailable() bool {
	playwrightOnce.Do(func() {
		if _, err := exec.LookPath("python3"); err != nil {
			return
		}
		cmd := exec.Command("python3", "-c", "import playwright; print('ok')")
		output, err := cmd.Output()
		playwrightAvailable = err == nil && strings.TrimSpace(string(output)) == "ok"
	})
	return playwrightAvailable
}

// RenderWithPlaywright uses Playwright (via Python subprocess) to render a page
// with full menu discovery (desktop + mobile viewports, clicking nav triggers).
// Returns the final HTML and all discovered link URLs.
func RenderWithPlaywright(ctx context.Context, pageURL string) (*PlaywrightResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-c", playwrightScript(), pageURL)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("playwright render failed: %w; stderr: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("playwright render failed: %w", err)
	}

	var result PlaywrightResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("playwright output parse failed: %w", err)
	}

	return &result, nil
}

func playwrightScript() string {
	return `
import sys, json
from playwright.sync_api import sync_playwright

url = sys.argv[1]

with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)

    result = {"html": "", "links": []}

    # Desktop viewport
    page = browser.new_page(viewport={"width": 1440, "height": 900})
    page.goto(url, wait_until="networkidle", timeout=30000)
    page.wait_for_timeout(2000)

    # Collect links before interaction
    initial_links = set(page.evaluate("""
        () => [...document.querySelectorAll('a[href]')].map(a => a.href)
    """))

    # Menu discovery: click nav triggers
    menu_labels = ["services", "products", "solutions", "resources", "company", "more", "explore"]

    # Click text-based triggers in header/nav
    for label in menu_labels:
        try:
            el = page.locator("header >> text=/" + label + "/i, nav >> text=/" + label + "/i").first
            if el.is_visible(timeout=500):
                el.click()
                page.wait_for_timeout(500)
        except:
            pass

    # Click buttons with common trigger patterns
    triggers = page.locator("button[aria-haspopup], [aria-expanded='false'], nav button, header button")
    count = triggers.count()
    for i in range(min(count, 20)):
        try:
            t = triggers.nth(i)
            if t.is_visible(timeout=200):
                t.click()
                page.wait_for_timeout(300)
        except:
            pass

    # Collect links after desktop interaction
    desktop_links = set(page.evaluate("""
        () => [...document.querySelectorAll('a[href]')].map(a => a.href)
    """))

    # Mobile viewport
    page.set_viewport_size({"width": 390, "height": 844})
    page.wait_for_timeout(500)

    # Click mobile menu triggers
    for selector in ["button:has-text('Menu')", "button[aria-label*='menu' i]", ".hamburger", ".menu-toggle"]:
        try:
            el = page.locator(selector).first
            if el.is_visible(timeout=500):
                el.click()
                page.wait_for_timeout(500)
                # Click sub-menus
                for label in menu_labels:
                    try:
                        sub = page.locator("text=/" + label + "/i").first
                        if sub.is_visible(timeout=300):
                            sub.click()
                            page.wait_for_timeout(300)
                    except:
                        pass
                break
        except:
            pass

    # Collect all links after all interactions
    all_links = set(page.evaluate("""
        () => [...document.querySelectorAll('a[href]')].map(a => a.href)
    """))

    # Get final HTML
    result["html"] = page.content()
    result["links"] = list(all_links)

    browser.close()

print(json.dumps(result))
`
}
