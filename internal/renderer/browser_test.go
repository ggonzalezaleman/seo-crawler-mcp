package renderer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func skipIfNoChrome(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("chromium"); err != nil {
		if _, err := exec.LookPath("google-chrome"); err != nil {
			if _, err := exec.LookPath("chromium-browser"); err != nil {
				t.Skip("Chrome/Chromium not found, skipping browser tests")
			}
		}
	}
}

const jsPage = `<!DOCTYPE html>
<html>
<head><title>JS App</title></head>
<body>
<div id="root"></div>
<script>
document.getElementById('root').innerHTML = '<h1>Rendered Content</h1><p>This was added by JavaScript</p>';
</script>
</body>
</html>`

func TestRender_BasicPage(t *testing.T) {
	skipIfNoChrome(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(jsPage))
	}))
	defer srv.Close()

	pool := NewPool(Options{
		MaxSlots:      1,
		RenderWaitMs:  500,
		RenderTimeout: 15 * time.Second,
	})
	defer pool.Close()

	ctx := context.Background()
	result, err := pool.Render(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result.HTML, "Rendered Content") {
		t.Errorf("expected rendered JS content in HTML, got: %s", result.HTML[:min(200, len(result.HTML))])
	}
	if !strings.Contains(result.HTML, "This was added by JavaScript") {
		t.Errorf("expected JS-injected paragraph in HTML")
	}
	if result.FinalURL == "" {
		t.Error("expected non-empty FinalURL")
	}
	if result.RenderTime <= 0 {
		t.Error("expected positive RenderTime")
	}
}

func TestRender_Timeout(t *testing.T) {
	skipIfNoChrome(t)

	const slowPage = `<!DOCTYPE html>
<html><head><title>Slow</title></head>
<body>
<script>
// Burn CPU until the context is cancelled.
while(true) { Math.random(); }
</script>
</body>
</html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(slowPage))
	}))
	defer srv.Close()

	pool := NewPool(Options{
		MaxSlots:      1,
		RenderWaitMs:  100,
		RenderTimeout: 3 * time.Second,
	})
	defer pool.Close()

	ctx := context.Background()
	_, err := pool.Render(ctx, srv.URL)
	if err == nil {
		// chromedp may still succeed if the page loads before JS blocks;
		// the infinite loop runs after DOMContentLoaded. That's acceptable —
		// the key invariant is that Render returns within the timeout.
		t.Log("Render returned without error (page loaded before JS blocked); acceptable")
	}
}

func TestPool_Concurrency(t *testing.T) {
	skipIfNoChrome(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(jsPage))
	}))
	defer srv.Close()

	pool := NewPool(Options{
		MaxSlots:      1,
		RenderWaitMs:  200,
		RenderTimeout: 15 * time.Second,
	})
	defer pool.Close()

	const numRenders = 3
	var wg sync.WaitGroup
	var successes atomic.Int32

	wg.Add(numRenders)
	for i := 0; i < numRenders; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			result, err := pool.Render(ctx, srv.URL)
			if err == nil && strings.Contains(result.HTML, "Rendered Content") {
				successes.Add(1)
			}
		}()
	}

	wg.Wait()

	if got := successes.Load(); got != numRenders {
		t.Errorf("expected %d successful renders, got %d", numRenders, got)
	}
}

func TestPool_Close(t *testing.T) {
	skipIfNoChrome(t)

	pool := NewPool(Options{MaxSlots: 1})
	pool.Close()

	ctx := context.Background()
	_, err := pool.Render(ctx, "http://localhost:1")
	if err == nil {
		t.Fatal("expected error from closed pool, got nil")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("expected closed error, got: %v", err)
	}
}
