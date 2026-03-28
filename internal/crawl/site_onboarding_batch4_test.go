package crawl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOnboardHost_RobotsUnreachable_PolicyAllow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/sitemap_index.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/llms.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	f := setupFetcher()
	onboarder := NewHostOnboarderWithPolicy(f, nil, 1000, testUserAgent, "allow")

	host := strings.TrimPrefix(ts.URL, "http://")
	info, err := onboarder.OnboardHost(context.Background(), "job-allow", host, "http")
	if err != nil {
		t.Fatalf("OnboardHost failed: %v", err)
	}

	// With allow policy, RobotsFile should be nil (allow all)
	if info.RobotsFile != nil {
		t.Error("expected RobotsFile to be nil (allow-all) for allow policy")
	}

	hasAllowEvent := false
	for _, e := range info.Events {
		if strings.Contains(e, "policy=allow") {
			hasAllowEvent = true
			break
		}
	}
	if !hasAllowEvent {
		t.Error("expected policy=allow event")
	}
}

func TestOnboardHost_RobotsUnreachable_PolicyDisallow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/sitemap_index.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/llms.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	f := setupFetcher()
	onboarder := NewHostOnboarderWithPolicy(f, nil, 1000, testUserAgent, "disallow")

	host := strings.TrimPrefix(ts.URL, "http://")
	info, err := onboarder.OnboardHost(context.Background(), "job-disallow", host, "http")
	if err != nil {
		t.Fatalf("OnboardHost failed: %v", err)
	}

	// With disallow policy, RobotsFile should block everything
	if info.RobotsFile == nil {
		t.Fatal("expected RobotsFile to be non-nil for disallow policy")
	}
	if info.RobotsFile.IsAllowed("*", "/anything") {
		t.Error("expected /anything to be disallowed")
	}

	hasDisallowEvent := false
	for _, e := range info.Events {
		if strings.Contains(e, "policy=disallow") {
			hasDisallowEvent = true
			break
		}
	}
	if !hasDisallowEvent {
		t.Error("expected policy=disallow event")
	}
}

func TestOnboardHost_RobotsUnreachable_PolicyCacheThenAllow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/sitemap_index.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/llms.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	f := setupFetcher()
	onboarder := NewHostOnboarderWithPolicy(f, nil, 1000, testUserAgent, "cache_then_allow")

	host := strings.TrimPrefix(ts.URL, "http://")
	info, err := onboarder.OnboardHost(context.Background(), "job-cache", host, "http")
	if err != nil {
		t.Fatalf("OnboardHost failed: %v", err)
	}

	// cache_then_allow without cache falls back to allow
	if info.RobotsFile != nil {
		t.Error("expected RobotsFile to be nil (fallback allow-all) for cache_then_allow without cache")
	}

	hasCacheEvent := false
	for _, e := range info.Events {
		if strings.Contains(e, "cache_then_allow") {
			hasCacheEvent = true
			break
		}
	}
	if !hasCacheEvent {
		t.Error("expected cache_then_allow event")
	}
}

func TestOnboardHost_RobotsRetrySuccess(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(500)
			return
		}
		// Second call succeeds
		fmt.Fprint(w, "User-agent: *\nDisallow: /secret/\n")
	})
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/sitemap_index.xml", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/llms.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	f := setupFetcher()
	onboarder := NewHostOnboarderWithPolicy(f, nil, 1000, testUserAgent, "allow")

	host := strings.TrimPrefix(ts.URL, "http://")
	info, err := onboarder.OnboardHost(context.Background(), "job-retry", host, "http")
	if err != nil {
		t.Fatalf("OnboardHost failed: %v", err)
	}

	// Retry succeeded, should have parsed robots
	if info.RobotsFile == nil {
		t.Fatal("expected RobotsFile to be non-nil after successful retry")
	}
	if info.RobotsFile.IsAllowed("*", "/secret/page") {
		t.Error("expected /secret/page to be disallowed after successful retry")
	}
}
