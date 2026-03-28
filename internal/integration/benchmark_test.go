package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/config"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/engine"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/fetcher"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/ssrf"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/storage"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/urlutil"
)

// newBenchmarkSite creates a programmatic fixture site with pageCount pages.
// Each page links to 3-5 other pages deterministically, creating a realistic
// crawl graph. The homepage links to the first 50 pages.
func newBenchmarkSite(pageCount int) *httptest.Server {
	mux := http.NewServeMux()

	// Pre-generate all page content to avoid closure issues and allocation during serving
	pages := make([]string, pageCount)
	for i := 0; i < pageCount; i++ {
		pages[i] = generateBenchPage(i, pageCount)
	}

	// Register page handlers
	for i := 0; i < pageCount; i++ {
		content := pages[i]
		path := fmt.Sprintf("/page/%d", i)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(content))
		})
	}

	// Homepage links to first 50 pages (or all if fewer)
	linkCount := pageCount
	if linkCount > 50 {
		linkCount = 50
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(404)
			w.Write([]byte("<html><body><h1>Not Found</h1></body></html>"))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		var sb strings.Builder
		sb.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">`)
		sb.WriteString(`<title>Benchmark Site Home</title>`)
		sb.WriteString(`<meta name="description" content="Home page of the benchmark fixture site with many pages">`)
		sb.WriteString(`</head><body><h1>Benchmark Site</h1><ul>`)
		for j := 0; j < linkCount; j++ {
			sb.WriteString(fmt.Sprintf(`<li><a href="/page/%d">Page %d</a></li>`, j, j))
		}
		sb.WriteString(`</ul></body></html>`)
		w.Write([]byte(sb.String()))
	})

	// robots.txt
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("User-agent: *\nAllow: /\n"))
	})

	return httptest.NewServer(mux)
}

// generateBenchPage creates a realistic HTML page with deterministic links,
// content, and optional JSON-LD.
func generateBenchPage(id, total int) string {
	rng := rand.New(rand.NewSource(int64(id)))

	// Determine 3-5 outbound links to other pages
	numLinks := 3 + rng.Intn(3) // 3, 4, or 5
	var links []int
	seen := map[int]bool{id: true} // avoid self-link
	for len(links) < numLinks {
		target := rng.Intn(total)
		if !seen[target] {
			seen[target] = true
			links = append(links, target)
		}
	}

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">`)
	sb.WriteString(fmt.Sprintf(`<title>Page %d - Benchmark Site</title>`, id))
	sb.WriteString(fmt.Sprintf(`<meta name="description" content="Benchmark page number %d with detailed content about topic %d">`, id, id%50))
	sb.WriteString(fmt.Sprintf(`<link rel="canonical" href="/page/%d">`, id))

	// Every 10th page gets JSON-LD
	if id%10 == 0 {
		sb.WriteString(fmt.Sprintf(`<script type="application/ld+json">
{"@context":"https://schema.org","@type":"Article","headline":"Page %d","author":{"@type":"Person","name":"Author %d"}}
</script>`, id, id%20))
	}

	sb.WriteString(`</head><body>`)
	sb.WriteString(fmt.Sprintf(`<h1>Benchmark Page %d</h1>`, id))

	// Generate 200-300 words of content (deterministic)
	wordCount := 200 + rng.Intn(101) // 200-300
	sb.WriteString("<p>")
	words := benchWords()
	for w := 0; w < wordCount; w++ {
		if w > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(words[rng.Intn(len(words))])
		if (w+1)%15 == 0 {
			sb.WriteString(". ")
		}
	}
	sb.WriteString("</p>")

	// Navigation links
	sb.WriteString("<nav><ul>")
	for _, target := range links {
		sb.WriteString(fmt.Sprintf(`<li><a href="/page/%d">Go to page %d</a></li>`, target, target))
	}
	sb.WriteString(`</ul></nav>`)
	sb.WriteString(`<a href="/">Home</a>`)
	sb.WriteString(`</body></html>`)

	return sb.String()
}

// benchWords returns a vocabulary for deterministic content generation.
func benchWords() []string {
	return []string{
		"search", "engine", "optimization", "website", "content", "ranking",
		"algorithm", "traffic", "organic", "digital", "marketing", "strategy",
		"performance", "analysis", "technical", "crawl", "index", "mobile",
		"responsive", "speed", "loading", "structure", "authority", "backlink",
		"internal", "linking", "meta", "description", "heading", "image",
		"schema", "markup", "quality", "user", "experience", "engagement",
		"bounce", "rate", "conversion", "keyword", "research", "competitor",
		"audit", "sitemap", "robots", "canonical", "redirect", "broken",
		"accessibility", "semantic", "structured", "data", "rich", "snippet",
		"featured", "local", "international", "hreflang", "pagination",
		"core", "web", "vitals", "lighthouse", "page", "domain",
	}
}

// runCrawl is a helper that sets up and runs a crawl against a test server.
func runBenchCrawl(t testing.TB, siteURL string, maxPages int) (*storage.CrawlJob, error) {
	dir := ""
	switch tb := t.(type) {
	case *testing.T:
		dir = tb.TempDir()
	case *testing.B:
		dir = tb.TempDir()
	}

	db, err := storage.Open(filepath.Join(dir, "bench.db"))
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	cfg := config.DefaultConfig()
	cfg.MaxPages = maxPages
	cfg.MaxDepth = 20
	cfg.GlobalConcurrency = 8
	cfg.PerHostConcurrency = 4
	cfg.RequestTimeout = 10 * time.Second
	cfg.RenderMode = config.RenderModeStatic
	cfg.AllowPrivateNetworks = true
	cfg.SSRFProtection = false

	guard := ssrf.NewGuard(true)
	f := fetcher.New(fetcher.Options{
		UserAgent:           "benchmark-test/1.0",
		Timeout:             10 * time.Second,
		MaxResponseBody:     5 * 1024 * 1024,
		MaxDecompressedBody: 20 * 1024 * 1024,
		MaxRedirectHops:     10,
		Retries:             0,
	})
	rl := fetcher.NewRateLimiter(cfg.PerHostConcurrency)

	tsURL, err := url.Parse(siteURL)
	if err != nil {
		return nil, fmt.Errorf("parsing site URL: %w", err)
	}
	sc, err := urlutil.NewScopeChecker("exact_host", tsURL.Hostname(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating scope checker: %w", err)
	}

	eng := engine.New(engine.EngineConfig{
		DB:           db,
		Fetcher:      f,
		RateLimiter:  rl,
		ScopeChecker: sc,
		SSRFGuard:    guard,
		Config:       &cfg,
	})

	seedURLs, _ := json.Marshal([]string{siteURL + "/"})
	job, err := db.CreateJob("crawl", "{}", string(seedURLs))
	if err != nil {
		return nil, fmt.Errorf("creating job: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := eng.RunCrawl(ctx, job.ID); err != nil {
		return nil, fmt.Errorf("RunCrawl failed: %w", err)
	}

	job, err = db.GetJob(job.ID)
	if err != nil {
		return nil, fmt.Errorf("getting job: %w", err)
	}

	return job, nil
}

// TestBenchmarkFixture verifies the programmatic benchmark site works correctly
// with a smaller 100-page crawl.
func TestBenchmarkFixture(t *testing.T) {
	site := newBenchmarkSite(100)
	defer site.Close()

	t.Logf("benchmark fixture site at %s (100 pages)", site.URL)

	job, err := runBenchCrawl(t, site.URL, 150) // allow more than 100 to account for homepage
	if err != nil {
		t.Fatalf("crawl failed: %v", err)
	}

	t.Logf("pages crawled: %d", job.PagesCrawled)

	if job.PagesCrawled < 50 {
		t.Errorf("expected at least 50 pages crawled, got %d", job.PagesCrawled)
	}

	if job.Status != "completed" {
		t.Errorf("expected job status 'completed', got %q", job.Status)
	}
}

// BenchmarkCrawl1000Pages measures the performance of crawling a 1000-page site.
func BenchmarkCrawl1000Pages(b *testing.B) {
	site := newBenchmarkSite(1000)
	defer site.Close()

	b.Logf("benchmark fixture site at %s (1000 pages)", site.URL)

	for i := 0; i < b.N; i++ {
		dir := b.TempDir()
		db, err := storage.Open(filepath.Join(dir, "bench.db"))
		if err != nil {
			b.Fatalf("opening database: %v", err)
		}

		cfg := config.DefaultConfig()
		cfg.MaxPages = 1100 // allow slightly more than 1000
		cfg.MaxDepth = 30
		cfg.GlobalConcurrency = 8
		cfg.PerHostConcurrency = 4
		cfg.RequestTimeout = 10 * time.Second
		cfg.RenderMode = config.RenderModeStatic
		cfg.AllowPrivateNetworks = true
		cfg.SSRFProtection = false

		guard := ssrf.NewGuard(true)
		f := fetcher.New(fetcher.Options{
			UserAgent:           "benchmark/1.0",
			Timeout:             10 * time.Second,
			MaxResponseBody:     5 * 1024 * 1024,
			MaxDecompressedBody: 20 * 1024 * 1024,
			MaxRedirectHops:     10,
			Retries:             0,
		})
		rl := fetcher.NewRateLimiter(cfg.PerHostConcurrency)

		tsURL, _ := url.Parse(site.URL)
		sc, _ := urlutil.NewScopeChecker("exact_host", tsURL.Hostname(), nil)

		eng := engine.New(engine.EngineConfig{
			DB:           db,
			Fetcher:      f,
			RateLimiter:  rl,
			ScopeChecker: sc,
			SSRFGuard:    guard,
			Config:       &cfg,
		})

		seedURLs, _ := json.Marshal([]string{site.URL + "/"})
		job, err := db.CreateJob("crawl", "{}", string(seedURLs))
		if err != nil {
			b.Fatalf("creating job: %v", err)
		}

		b.ResetTimer()
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		err = eng.RunCrawl(ctx, job.ID)
		b.StopTimer()
		cancel()

		if err != nil {
			b.Fatalf("RunCrawl failed: %v", err)
		}

		// Report metrics
		job, err = db.GetJob(job.ID)
		if err != nil {
			b.Fatalf("getting job: %v", err)
		}
		b.ReportMetric(float64(job.PagesCrawled), "pages")

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		b.ReportMetric(float64(m.Alloc)/1024/1024, "MB-alloc")

		b.Logf("iteration %d: crawled %d pages, alloc %.1f MB", i, job.PagesCrawled, float64(m.Alloc)/1024/1024)

		db.Close()
	}
}
