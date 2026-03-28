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

// testSiteCrossScope serves a site where one page redirects to an external domain.
func testSiteCrossScope(externalURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Home</title>
				<meta name="description" content="This is the home page with enough content to pass threshold checks easily."></head>
				<body><h1>Home</h1>
				<p>Welcome to our test site. This page has enough content to avoid thin content detection during testing.</p>
				<a href="/redirect-page">Redirect Page</a></body></html>`)
		case "/redirect-page":
			// Redirect to external domain
			http.Redirect(w, r, externalURL+"/external-landing", http.StatusMovedPermanently)
		default:
			w.WriteHeader(404)
		}
	})
}

func TestCrossScopeRedirect_OutOfScopeStatus(t *testing.T) {
	// External server
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>External</title></head><body><h1>External</h1><p>Content here.</p></body></html>`)
	}))
	defer external.Close()

	// Internal server that redirects to external
	internal := httptest.NewServer(testSiteCrossScope(external.URL))
	defer internal.Close()

	dbPath := filepath.Join(t.TempDir(), "test-cross-scope.db")
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

	f := fetcher.New(fetcher.Options{
		UserAgent:           "test-crawler/1.0",
		Timeout:             5 * time.Second,
		MaxResponseBody:     5 * 1024 * 1024,
		MaxDecompressedBody: 20 * 1024 * 1024,
		MaxRedirectHops:     10,
	})
	rl := fetcher.NewRateLimiter(cfg.PerHostConcurrency)

	tsURL, _ := url.Parse(internal.URL)
	sc, _ := urlutil.NewScopeChecker("exact_host", tsURL.Hostname(), nil)

	seedURLs, _ := json.Marshal([]string{internal.URL + "/"})
	job, err := db.CreateJob("crawl", "{}", string(seedURLs))
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}

	eng := New(EngineConfig{
		DB:           db,
		Fetcher:      f,
		RateLimiter:  rl,
		ScopeChecker: sc,
		Config:       &cfg,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := eng.RunCrawl(ctx, job.ID); err != nil {
		t.Fatalf("RunCrawl: %v", err)
	}

	// Verify the external URL got status "out_of_scope"
	externalNorm, _ := urlutil.Normalize(external.URL + "/external-landing")
	extURL, err := db.GetURLByNormalized(job.ID, externalNorm)
	if err != nil {
		t.Fatalf("external URL not found in DB: %v", err)
	}
	if extURL.Status != "out_of_scope" {
		t.Errorf("external URL status = %q, want out_of_scope", extURL.Status)
	}
}

// testSiteWithExternalCanonical serves a page with an external canonical.
func testSiteWithExternalCanonical(externalURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Home Page Title</title>
				<meta name="description" content="This is the home page with a canonical pointing to an external domain for testing.">
				<link rel="canonical" href="%s/canonical-target">
				</head>
				<body><h1>Home</h1>
				<p>This page has an external canonical URL that should trigger a HEAD request during crawling.</p></body></html>`, externalURL)
		default:
			w.WriteHeader(404)
		}
	})
}

func TestHeadRequestForOutOfScopeCanonical(t *testing.T) {
	headReceived := false

	// External server that tracks HEAD requests
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead && r.URL.Path == "/canonical-target" {
			headReceived = true
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(200)
	}))
	defer external.Close()

	internal := httptest.NewServer(testSiteWithExternalCanonical(external.URL))
	defer internal.Close()

	dbPath := filepath.Join(t.TempDir(), "test-head-canonical.db")
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

	f := fetcher.New(fetcher.Options{
		UserAgent:           "test-crawler/1.0",
		Timeout:             5 * time.Second,
		MaxResponseBody:     5 * 1024 * 1024,
		MaxDecompressedBody: 20 * 1024 * 1024,
		MaxRedirectHops:     10,
	})
	rl := fetcher.NewRateLimiter(cfg.PerHostConcurrency)

	tsURL, _ := url.Parse(internal.URL)
	sc, _ := urlutil.NewScopeChecker("exact_host", tsURL.Hostname(), nil)

	seedURLs, _ := json.Marshal([]string{internal.URL + "/"})
	job, err := db.CreateJob("crawl", "{}", string(seedURLs))
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}

	eng := New(EngineConfig{
		DB:           db,
		Fetcher:      f,
		RateLimiter:  rl,
		ScopeChecker: sc,
		Config:       &cfg,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := eng.RunCrawl(ctx, job.ID); err != nil {
		t.Fatalf("RunCrawl: %v", err)
	}

	if !headReceived {
		t.Error("expected HEAD request to be sent to out-of-scope canonical target")
	}

	// Verify the edge has target_status_code set
	rootNorm, _ := urlutil.Normalize(internal.URL + "/")
	rootURL, err := db.GetURLByNormalized(job.ID, rootNorm)
	if err != nil {
		t.Fatalf("root URL not found: %v", err)
	}

	edges, err := db.GetEdgesBySource(job.ID, rootURL.ID, 100, "")
	if err != nil {
		t.Fatalf("getting edges: %v", err)
	}

	foundCanonical := false
	for _, edge := range edges {
		if edge.RelationType.Valid && edge.RelationType.String == "canonical" {
			foundCanonical = true
			if !edge.TargetStatusCode.Valid {
				t.Error("canonical edge should have target_status_code set")
			} else if edge.TargetStatusCode.Int64 != 200 {
				t.Errorf("canonical edge target_status_code = %d, want 200", edge.TargetStatusCode.Int64)
			}
		}
	}
	if !foundCanonical {
		t.Error("expected to find a canonical edge")
	}
}

func TestForceRenderPatterns_EngineMarksJSSuspect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Home</title>
				<meta name="description" content="This home page has enough content to avoid thin content detection during testing."></head>
				<body><h1>Home</h1>
				<p>Enough content to be fine with our low threshold for testing purposes in this integration test.</p>
				<a href="/app/dashboard">Dashboard</a></body></html>`)
		case "/app/dashboard":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			// This page has NO JS frameworks, but forceRenderPatterns should mark it
			fmt.Fprint(w, `<!DOCTYPE html><html><head><title>Dashboard</title>
				<meta name="description" content="This dashboard page should be marked for browser rendering via forceRenderPatterns."></head>
				<body><h1>Dashboard</h1>
				<p>This is a dashboard page. It should be marked as JS suspect due to forceRenderPatterns configuration.</p></body></html>`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer ts.Close()

	dbPath := filepath.Join(t.TempDir(), "test-force-render.db")
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
	cfg.ForceRenderPatterns = []string{"/app/*"}

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
	job, err := db.CreateJob("crawl", "{}", string(seedURLs))
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}

	eng := New(EngineConfig{
		DB:           db,
		Fetcher:      f,
		RateLimiter:  rl,
		ScopeChecker: sc,
		Config:       &cfg,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := eng.RunCrawl(ctx, job.ID); err != nil {
		t.Fatalf("RunCrawl: %v", err)
	}

	// Verify /app/dashboard page has js_suspect=1
	dashNorm, _ := urlutil.Normalize(ts.URL + "/app/dashboard")
	dashURL, err := db.GetURLByNormalized(job.ID, dashNorm)
	if err != nil {
		t.Fatalf("dashboard URL not found: %v", err)
	}

	dashPage, err := db.GetPageByURL(job.ID, dashURL.ID)
	if err != nil {
		t.Fatalf("dashboard page not found: %v", err)
	}

	if dashPage.JSSuspect != 1 {
		t.Errorf("dashboard page js_suspect = %d, want 1 (forced by forceRenderPatterns)", dashPage.JSSuspect)
	}

	// Verify / page does NOT have js_suspect forced
	rootNorm, _ := urlutil.Normalize(ts.URL + "/")
	rootURL, err := db.GetURLByNormalized(job.ID, rootNorm)
	if err != nil {
		t.Fatalf("root URL not found: %v", err)
	}

	rootPage, err := db.GetPageByURL(job.ID, rootURL.ID)
	if err != nil {
		t.Fatalf("root page not found: %v", err)
	}

	if rootPage.JSSuspect != 0 {
		t.Errorf("root page js_suspect = %d, want 0 (not in forceRenderPatterns)", rootPage.JSSuspect)
	}
}
