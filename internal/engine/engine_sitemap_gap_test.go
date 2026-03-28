package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/config"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/fetcher"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/storage"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/urlutil"
)

// TestSitemapGapDetection verifies that sitemapGapEscalation correctly identifies
// sitemap URLs that have no inbound static HTML links.
func TestSitemapGapDetection(t *testing.T) {
	// Setup: a site where /about is linked from / but /hidden and /secret are only in the sitemap.
	pages := map[string]string{
		"/": `<!DOCTYPE html><html><head><title>Home</title>
			<meta name="description" content="Home page with enough words to avoid thin content."></head>
			<body><h1>Home</h1>
			<p>Welcome to the test site with enough content to pass thin content checks during testing.</p>
			<a href="/about">About</a></body></html>`,

		"/about": `<!DOCTYPE html><html><head><title>About</title>
			<meta name="description" content="About page with enough words to avoid thin content."></head>
			<body><h1>About</h1>
			<p>This is the about page with enough content words to pass thin content thresholds.</p></body></html>`,

		"/hidden": `<!DOCTYPE html><html><head><title>Hidden</title>
			<meta name="description" content="Hidden page only in sitemap not linked anywhere."></head>
			<body><h1>Hidden</h1>
			<p>This page is only discoverable via the sitemap and not linked from any other page.</p></body></html>`,

		"/secret": `<!DOCTYPE html><html><head><title>Secret</title>
			<meta name="description" content="Secret page only in sitemap not linked anywhere."></head>
			<body><h1>Secret</h1>
			<p>This page is also only discoverable via the sitemap and not linked from any other page.</p></body></html>`,

		"/sitemap.xml": "", // handled below
		"/robots.txt":  "", // handled below
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", "http://"+r.Host)
			return
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>%s/</loc></url>
  <url><loc>%s/about</loc></url>
  <url><loc>%s/hidden</loc></url>
  <url><loc>%s/secret</loc></url>
</urlset>`, "http://"+r.Host, "http://"+r.Host, "http://"+r.Host, "http://"+r.Host)
			return
		}
		html, ok := pages[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}))
	defer ts.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	cfg := config.DefaultConfig()
	cfg.GlobalConcurrency = 2
	cfg.MaxPages = 100
	cfg.MaxDepth = 10
	cfg.RequestTimeout = 5 * time.Second
	cfg.AllowPrivateNetworks = true
	cfg.SSRFProtection = false
	cfg.ThinContentThreshold = 10
	cfg.MaxQueryVariantsPerPath = 50
	// Set hybrid mode so the escalation code path is triggered
	cfg.RenderMode = config.RenderModeHybrid

	f := fetcher.New(fetcher.Options{
		UserAgent:           "test-crawler/1.0",
		Timeout:             5 * time.Second,
		MaxResponseBody:     5 * 1024 * 1024,
		MaxDecompressedBody: 20 * 1024 * 1024,
		MaxRedirectHops:     10,
		SSRFGuard:           nil,
	})
	rl := fetcher.NewRateLimiter(cfg.PerHostConcurrency)

	tsURL, _ := url.Parse(ts.URL)
	sc, err := urlutil.NewScopeChecker("exact_host", tsURL.Hostname(), nil)
	if err != nil {
		t.Fatalf("creating scope checker: %v", err)
	}

	seedURLs, _ := json.Marshal([]string{ts.URL + "/"})
	job, err := db.CreateJob("crawl", "{}", string(seedURLs))
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}

	// Run the crawl with NO renderer (should log gap but not escalate)
	eng := New(EngineConfig{
		DB:           db,
		Fetcher:      f,
		RateLimiter:  rl,
		ScopeChecker: sc,
		Config:       &cfg,
		Renderer:     nil, // no renderer
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := eng.RunCrawl(ctx, job.ID); err != nil {
		t.Fatalf("RunCrawl: %v", err)
	}

	// Verify the crawl completed
	job, err = db.GetJob(job.ID)
	if err != nil {
		t.Fatalf("getting job: %v", err)
	}
	if job.Status != "completed" {
		t.Fatalf("job status = %q, want completed", job.Status)
	}

	// Verify that /hidden and /secret were crawled (they're in the sitemap so added to frontier)
	// but they should have NO inbound link edges from static HTML
	hiddenNorm, _ := urlutil.Normalize(ts.URL + "/hidden")
	secretNorm, _ := urlutil.Normalize(ts.URL + "/secret")

	// Check that /hidden has no inbound link edges from static crawl
	var hiddenInboundLinks int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM edges e
		 JOIN urls u ON u.id = e.normalized_target_url_id
		 WHERE e.job_id = ? AND u.normalized_url = ? AND e.discovery_mode = 'static' AND e.relation_type = 'link'`,
		job.ID, hiddenNorm,
	).Scan(&hiddenInboundLinks)
	if err != nil {
		t.Fatalf("querying hidden inbound links: %v", err)
	}
	if hiddenInboundLinks != 0 {
		t.Errorf("/hidden has %d inbound static links, want 0", hiddenInboundLinks)
	}

	// Same for /secret
	var secretInboundLinks int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM edges e
		 JOIN urls u ON u.id = e.normalized_target_url_id
		 WHERE e.job_id = ? AND u.normalized_url = ? AND e.discovery_mode = 'static' AND e.relation_type = 'link'`,
		job.ID, secretNorm,
	).Scan(&secretInboundLinks)
	if err != nil {
		t.Fatalf("querying secret inbound links: %v", err)
	}
	if secretInboundLinks != 0 {
		t.Errorf("/secret has %d inbound static links, want 0", secretInboundLinks)
	}

	// Verify the sitemap_gap_escalation event was logged
	var eventCount int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM crawl_events WHERE job_id = ? AND event_type = 'sitemap_gap_escalation'`,
		job.ID,
	).Scan(&eventCount)
	if err != nil {
		t.Fatalf("querying escalation events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("sitemap_gap_escalation events = %d, want 1", eventCount)
	}

	// Verify the event details indicate no renderer was available
	var detailsJSON string
	err = db.QueryRow(
		`SELECT details_json FROM crawl_events WHERE job_id = ? AND event_type = 'sitemap_gap_escalation'`,
		job.ID,
	).Scan(&detailsJSON)
	if err != nil {
		t.Fatalf("querying escalation event details: %v", err)
	}

	var details struct {
		GapCount        int    `json:"gapCount"`
		PagesReRendered int    `json:"pagesReRendered"`
		NewLinksFound   int    `json:"newLinksFound"`
		NewURLsQueued   int    `json:"newURLsQueued"`
		Reason          string `json:"reason"`
	}
	if jsonErr := json.Unmarshal([]byte(detailsJSON), &details); jsonErr != nil {
		t.Fatalf("parsing event details: %v", jsonErr)
	}

	// /hidden and /secret should be in the gap
	if details.GapCount < 2 {
		t.Errorf("gapCount = %d, want >= 2 (at least /hidden and /secret)", details.GapCount)
	}
	if details.Reason != "no_renderer" {
		t.Errorf("reason = %q, want no_renderer", details.Reason)
	}
	if details.PagesReRendered != 0 {
		t.Errorf("pagesReRendered = %d, want 0 (no renderer)", details.PagesReRendered)
	}
}

// TestSitemapGapNoGap verifies no escalation event when all sitemap URLs are linked.
func TestSitemapGapNoGap(t *testing.T) {
	// All sitemap URLs are also linked from static HTML
	pages := map[string]string{
		"/": `<!DOCTYPE html><html><head><title>Home</title>
			<meta name="description" content="Home page with links to all pages."></head>
			<body><h1>Home</h1>
			<p>Welcome to the test site with enough content words to pass thin content checks.</p>
			<a href="/about">About</a></body></html>`,

		"/about": `<!DOCTYPE html><html><head><title>About</title>
			<meta name="description" content="About page linked from home."></head>
			<body><h1>About</h1>
			<p>About page with enough content words to pass thin content threshold checks.</p>
			<a href="/">Home</a></body></html>`,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", "http://"+r.Host)
			return
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>%s/</loc></url>
  <url><loc>%s/about</loc></url>
</urlset>`, "http://"+r.Host, "http://"+r.Host)
			return
		}
		html, ok := pages[r.URL.Path]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}))
	defer ts.Close()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	cfg := config.DefaultConfig()
	cfg.GlobalConcurrency = 2
	cfg.MaxPages = 100
	cfg.MaxDepth = 10
	cfg.RequestTimeout = 5 * time.Second
	cfg.AllowPrivateNetworks = true
	cfg.SSRFProtection = false
	cfg.ThinContentThreshold = 10
	cfg.RenderMode = config.RenderModeHybrid

	f := fetcher.New(fetcher.Options{
		UserAgent:           "test-crawler/1.0",
		Timeout:             5 * time.Second,
		MaxResponseBody:     5 * 1024 * 1024,
		MaxDecompressedBody: 20 * 1024 * 1024,
		MaxRedirectHops:     10,
	})
	rl := fetcher.NewRateLimiter(cfg.PerHostConcurrency)

	tsURL, _ := url.Parse(ts.URL)
	sc, _ := urlutil.NewScopeChecker("exact_host", tsURL.Hostname(), nil)

	seedURLs, _ := json.Marshal([]string{ts.URL + "/"})
	job, _ := db.CreateJob("crawl", "{}", string(seedURLs))

	eng := New(EngineConfig{
		DB:          db,
		Fetcher:     f,
		RateLimiter: rl,
		ScopeChecker: sc,
		Config:      &cfg,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := eng.RunCrawl(ctx, job.ID); err != nil {
		t.Fatalf("RunCrawl: %v", err)
	}

	// Verify NO sitemap_gap_escalation event (all URLs are linked)
	var eventCount int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM crawl_events WHERE job_id = ? AND event_type = 'sitemap_gap_escalation'`,
		job.ID,
	).Scan(&eventCount)
	if err != nil {
		t.Fatalf("querying escalation events: %v", err)
	}
	if eventCount != 0 {
		t.Errorf("sitemap_gap_escalation events = %d, want 0 (no gap)", eventCount)
	}
}
