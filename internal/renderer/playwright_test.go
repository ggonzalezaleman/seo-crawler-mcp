package renderer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func skipIfNoPlaywright(t *testing.T) {
	t.Helper()
	if !IsPlaywrightAvailable() {
		t.Skip("Playwright (python3 + playwright package) not available, skipping")
	}
}

// hiddenMenuPage serves a page with a hamburger button that reveals nav links
// only when clicked — exactly the scenario Playwright should handle.
const hiddenMenuPage = `<!DOCTYPE html>
<html>
<head><title>Menu Test</title></head>
<body>
<header>
  <button aria-label="menu" onclick="document.getElementById('menu').style.display='block'">☰ Menu</button>
</header>
<nav id="menu" style="display:none">
  <a href="/services">Services</a>
  <a href="/about">About</a>
  <a href="/contact">Contact</a>
</nav>
<main>
  <a href="/visible-link">Visible Link</a>
</main>
</body>
</html>`

func TestRenderWithPlaywright_MenuDiscovery(t *testing.T) {
	skipIfNoPlaywright(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(hiddenMenuPage))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := RenderWithPlaywright(ctx, srv.URL)
	if err != nil {
		t.Fatalf("RenderWithPlaywright failed: %v", err)
	}

	// Should have discovered links
	if len(result.Links) == 0 {
		t.Fatal("expected discovered links, got none")
	}

	// The visible link should always be found
	foundVisible := false
	for _, link := range result.Links {
		if strings.HasSuffix(link, "/visible-link") {
			foundVisible = true
			break
		}
	}
	if !foundVisible {
		t.Error("expected /visible-link in discovered links")
	}

	// HTML should be non-empty
	if len(result.HTML) == 0 {
		t.Error("expected non-empty HTML")
	}

	// HTML should contain the nav links (they're in the DOM regardless of visibility)
	for _, href := range []string{"/services", "/about", "/contact"} {
		if !strings.Contains(result.HTML, href) {
			t.Errorf("expected %q in HTML", href)
		}
	}
}

func TestRenderWithPlaywright_BasicPage(t *testing.T) {
	skipIfNoPlaywright(t)

	const basicPage = `<!DOCTYPE html>
<html>
<head><title>Basic</title></head>
<body>
<a href="/page1">Page 1</a>
<a href="/page2">Page 2</a>
</body>
</html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(basicPage))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := RenderWithPlaywright(ctx, srv.URL)
	if err != nil {
		t.Fatalf("RenderWithPlaywright failed: %v", err)
	}

	if len(result.Links) < 2 {
		t.Errorf("expected at least 2 links, got %d", len(result.Links))
	}

	if !strings.Contains(result.HTML, "Page 1") {
		t.Error("expected 'Page 1' in HTML")
	}
}

func TestIsPlaywrightAvailable(t *testing.T) {
	// This just tests that the function doesn't panic and returns a bool.
	available := IsPlaywrightAvailable()
	t.Logf("Playwright available: %v", available)
}
