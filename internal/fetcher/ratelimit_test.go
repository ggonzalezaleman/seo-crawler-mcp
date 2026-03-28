package fetcher

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiter_Concurrency(t *testing.T) {
	rl := NewRateLimiter(2)
	host := "example.com"

	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.Acquire(host)
			defer rl.Release(host)

			cur := concurrent.Add(1)
			for {
				old := maxConcurrent.Load()
				if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			concurrent.Add(-1)
		}()
	}

	wg.Wait()

	got := maxConcurrent.Load()
	if got > 2 {
		t.Errorf("max concurrent = %d, want <= 2", got)
	}
	if got < 1 {
		t.Errorf("max concurrent = %d, want >= 1", got)
	}
}

func TestRateLimiter_CrawlDelay(t *testing.T) {
	rl := NewRateLimiter(1)
	host := "slow.example.com"
	rl.SetCrawlDelay(host, 100*time.Millisecond)

	rl.Acquire(host)
	rl.Release(host)

	start := time.Now()
	rl.Acquire(host)
	rl.Release(host)
	elapsed := time.Since(start)

	if elapsed < 80*time.Millisecond {
		t.Errorf("elapsed = %v, want >= ~100ms (crawl delay)", elapsed)
	}
}
