// Package engine orchestrates the SEO crawl pipeline.
package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/config"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/crawl"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/fetcher"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/frontier"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/issues"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/parser"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/robots"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/ssrf"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/storage"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/urlutil"
)

// EngineConfig holds all dependencies for the crawl engine.
type EngineConfig struct {
	DB           *storage.DB
	Fetcher      *fetcher.Fetcher
	RateLimiter  *fetcher.RateLimiter
	ScopeChecker *urlutil.ScopeChecker
	SSRFGuard    *ssrf.Guard
	Config       *config.Config
}

// Engine orchestrates a complete crawl pipeline.
type Engine struct {
	db           *storage.DB
	fetcher      *fetcher.Fetcher
	rateLimiter  *fetcher.RateLimiter
	scopeChecker *urlutil.ScopeChecker
	ssrfGuard    *ssrf.Guard
	config       *config.Config

	// robotsRules caches parsed robots.txt per host during a crawl.
	robotsRules   map[string]*robots.RobotsFile
	robotsRulesMu sync.RWMutex
}

// New creates a new crawl engine.
func New(cfg EngineConfig) *Engine {
	return &Engine{
		db:           cfg.DB,
		fetcher:      cfg.Fetcher,
		rateLimiter:  cfg.RateLimiter,
		scopeChecker: cfg.ScopeChecker,
		ssrfGuard:    cfg.SSRFGuard,
		config:       cfg.Config,
	}
}

type fetchResult struct {
	urlID    int64
	url      string
	host     string
	depth    int
	fetchSeq int
	result   *fetcher.FetchResult
	err      error
}

// discoveredImage holds a resolved image URL and its source page URL ID.
type discoveredImage struct {
	normalizedURL string
	host          string
	isInternal    bool
	sourceURLID   int64
}

type parseResult struct {
	fetchResult
	page   *parser.ParseResult
	edges  []crawl.DiscoveredEdge
	issues []issues.DetectedIssue
	images []discoveredImage
}

type persistItem struct {
	parseResult
	fetchSeq int
}

// RunCrawl executes a full crawl job. Blocks until complete or cancelled.
func (e *Engine) RunCrawl(ctx context.Context, jobID string) error {
	// 1. Init: load job, update status
	job, err := e.db.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("loading job: %w", err)
	}
	if job.Status != "queued" {
		return fmt.Errorf("job %s has status %q, expected queued", jobID, job.Status)
	}
	if err := e.db.UpdateJobStarted(jobID); err != nil {
		return fmt.Errorf("starting job: %w", err)
	}

	// 2. Purge expired analyze jobs
	e.db.PurgeExpiredAnalyzeJobs()

	// 3. Seed: parse seed URLs, normalize, upsert, push to frontier
	var seeds []string
	if err := json.Unmarshal([]byte(job.SeedURLs), &seeds); err != nil {
		return e.failJob(jobID, fmt.Errorf("parsing seed URLs: %w", err))
	}

	q := frontier.New()

	// Track query variant counts per path for crawl trap detection
	var queryVariantsMu sync.Mutex
	queryVariants := map[string]int{}

	// Track pages crawled for MaxPages limit
	var pagesCrawled atomic.Int64

	for _, seedURL := range seeds {
		normalized, err := urlutil.Normalize(seedURL)
		if err != nil {
			log.Printf("engine: skipping invalid seed URL %q: %v", seedURL, err)
			continue
		}
		parsed, err := url.Parse(normalized)
		if err != nil {
			continue
		}
		host := parsed.Hostname()
		urlID, err := e.db.UpsertURL(jobID, normalized, host, "queued", true, "seed")
		if err != nil {
			return e.failJob(jobID, fmt.Errorf("upserting seed URL: %w", err))
		}
		q.Push(frontier.Item{
			URLID:         urlID,
			NormalizedURL: normalized,
			Host:          host,
			Depth:         0,
		})
	}

	if q.Len() == 0 {
		return e.failJob(jobID, fmt.Errorf("no valid seed URLs"))
	}

	// Create scope checker from first seed if not provided via config.
	if e.scopeChecker == nil {
		first := q.Peek()
		scopeMode := "registrable_domain"
		var allowedHosts []string
		if e.config != nil {
			scopeMode = string(e.config.ScopeMode)
			allowedHosts = e.config.AllowedHosts
		}
		// Try to parse from job config_json as well.
		var jobCfg struct {
			ScopeMode    string   `json:"scopeMode"`
			AllowedHosts []string `json:"allowedHosts"`
		}
		if err := json.Unmarshal([]byte(job.ConfigJSON), &jobCfg); err == nil {
			if jobCfg.ScopeMode != "" {
				scopeMode = jobCfg.ScopeMode
			}
			if len(jobCfg.AllowedHosts) > 0 {
				allowedHosts = jobCfg.AllowedHosts
			}
		}
		sc, err := urlutil.NewScopeChecker(scopeMode, first.Host, allowedHosts)
		if err != nil {
			return e.failJob(jobID, fmt.Errorf("creating scope checker: %w", err))
		}
		e.scopeChecker = sc
	}

	// ---- Host onboarding: robots.txt + sitemap discovery ----
	e.robotsRules = map[string]*robots.RobotsFile{}
	{
		userAgent := e.config.UserAgent
		if userAgent == "" {
			userAgent = "seo-crawler-mcp/0.1"
		}
		sitemapMax := e.config.MaxSitemapEntries
		if sitemapMax <= 0 {
			sitemapMax = 500000
		}
		robotsPolicy := string(e.config.RobotsUnreachablePolicy)
		if robotsPolicy == "" {
			robotsPolicy = "allow"
		}
		onboarder := crawl.NewHostOnboarderWithPolicy(e.fetcher, e.db, sitemapMax, userAgent, robotsPolicy)

		seenHosts := map[string]bool{}
		for _, seedURL := range seeds {
			parsed, parseErr := url.Parse(seedURL)
			if parseErr != nil {
				continue
			}
			// Use parsed.Host (includes port) for URL construction in onboarding.
			hostWithPort := parsed.Host
			hostOnly := parsed.Hostname()
			if seenHosts[hostWithPort] {
				continue
			}
			seenHosts[hostWithPort] = true

			info, onboardErr := onboarder.OnboardHost(ctx, jobID, hostWithPort, parsed.Scheme)
			if onboardErr != nil {
				log.Printf("engine: onboarding host %q: %v", hostWithPort, onboardErr)
				continue
			}

			// Cache robots rules for fetch-time checking (keyed by hostname without port)
			if info.RobotsFile != nil {
				e.robotsRulesMu.Lock()
				e.robotsRules[hostOnly] = info.RobotsFile
				e.robotsRulesMu.Unlock()
			}

			// Apply crawl delay to rate limiter (keyed by hostname as used in fetcher)
			if info.CrawlDelay > 0 {
				e.rateLimiter.SetCrawlDelay(hostOnly, info.CrawlDelay)
				log.Printf("engine: crawl-delay for %q set to %v", hostOnly, info.CrawlDelay)
			}

			// Add sitemap URLs to frontier
			for _, entry := range info.SitemapEntries {
				normalized, normErr := urlutil.Normalize(entry.Loc)
				if normErr != nil {
					continue
				}
				parsedURL, parseErr2 := url.Parse(normalized)
				if parseErr2 != nil {
					continue
				}
				entryHost := parsedURL.Hostname()

				// Check scope
				if e.scopeChecker != nil && !e.scopeChecker.IsInScope(normalized) {
					continue
				}

				// Check robots rules before adding
				if e.config.RespectRobots && info.RobotsFile != nil {
					if !info.RobotsFile.IsAllowed(userAgent, parsedURL.Path) {
						continue
					}
				}

				urlID, upsertErr := e.db.UpsertURL(jobID, normalized, entryHost, "queued", true, "sitemap")
				if upsertErr != nil {
					continue
				}

				if !q.Contains(urlID) {
					q.Push(frontier.Item{
						URLID:         urlID,
						NormalizedURL: normalized,
						Host:          entryHost,
						Depth:         1, // sitemap URLs are depth 1
					})
				}
			}

			// Log onboarding event
			eventDetails := fmt.Sprintf(`{"host":%q,"sitemapEntries":%d,"crawlDelay":%q}`,
				hostWithPort, len(info.SitemapEntries), info.CrawlDelay.String())
			e.db.InsertEvent(jobID, "host_onboarded", &eventDetails, nil)
		}
	}

	// Channels
	// fetchQueue feeds items from the dispatcher to fetcher workers.
	fetchQueue := make(chan frontier.Item, 64)
	fetchResults := make(chan fetchResult, 64)
	persistQueue := make(chan persistItem, 128)

	// Monotonic fetch sequence counter
	var fetchSeq atomic.Int64

	// Concurrency
	concurrency := e.config.GlobalConcurrency
	if concurrency < 1 {
		concurrency = 8
	}
	parserCount := 4

	// Use context.WithCancelCause so any fatal error (e.g. persister) can unwind the whole pipeline.
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// Counters for job stats
	var urlsDiscovered atomic.Int64
	urlsDiscovered.Store(int64(q.Len()))
	var issuesFound atomic.Int64

	// inFlight tracks items between dispatch and persist completion.
	var inFlight atomic.Int64

	// ---- Persister (1 goroutine) ----
	var persisterWg sync.WaitGroup
	persisterWg.Add(1)
	go func() {
		defer persisterWg.Done()
		for item := range persistQueue {
			if pErr := e.persistItem(ctx, jobID, item); pErr != nil {
				var lastErr error
				for attempt := 1; attempt <= 3; attempt++ {
					time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
					lastErr = e.persistItem(ctx, jobID, item)
					if lastErr == nil {
						break
					}
				}
				if lastErr != nil {
					inFlight.Add(-1)
					cancel(fmt.Errorf("persist failed after retries: %w", lastErr))
					return
				}
			}
			inFlight.Add(-1)
		}
	}()

	// ---- Parser Pool ----
	var parserWg sync.WaitGroup
	for range parserCount {
		parserWg.Add(1)
		go func() {
			defer parserWg.Done()
			for fr := range fetchResults {
				pr := e.processParseResult(ctx, jobID, fr, q, &pagesCrawled, &urlsDiscovered, &queryVariantsMu, queryVariants)
				issuesFound.Add(int64(len(pr.issues)))
				persistQueue <- persistItem{
					parseResult: pr,
					fetchSeq:    fr.fetchSeq,
				}
			}
		}()
	}

	// ---- Fetcher Pool ----
	var fetcherWg sync.WaitGroup
	for range concurrency {
		fetcherWg.Add(1)
		go func() {
			defer fetcherWg.Done()
			for item := range fetchQueue {
				// Check scope
				if !e.scopeChecker.IsInScope(item.NormalizedURL) {
					inFlight.Add(-1)
					continue
				}

				// Check robots.txt rules
				if e.config.RespectRobots {
					parsedItem, parseErr := url.Parse(item.NormalizedURL)
					if parseErr == nil {
						e.robotsRulesMu.RLock()
						rf := e.robotsRules[item.Host]
						e.robotsRulesMu.RUnlock()
						if rf != nil && !rf.IsAllowed(e.config.UserAgent, parsedItem.Path) {
							e.db.UpdateURLStatus(item.URLID, "robots_blocked")
							inFlight.Add(-1)
							continue
						}
					}
				}

				// Acquire rate limiter
				e.rateLimiter.Acquire(item.Host)

				// Get fetch sequence (must be unique per fetch)
				seq := int(fetchSeq.Add(1))

				// Fetch
				result, fetchErr := e.fetcher.Fetch(item.NormalizedURL)

				// Release rate limiter
				e.rateLimiter.Release(item.Host)

				// Track TTFB for slow-host detection
				if result != nil {
					if avgTTFB, full := e.rateLimiter.RecordTTFB(item.Host, result.TTFBMS); full && avgTTFB > 5000 {
						detailsJSON := fmt.Sprintf(`{"host":%q,"avgTtfbMs":%d}`, item.Host, avgTTFB)
						e.db.InsertIssue(storage.IssueInput{
							JobID:       jobID,
							IssueType:   "slow_host",
							Severity:    "info",
							Scope:       "page_local",
							DetailsJSON: &detailsJSON,
						})
					}
				}

				fr := fetchResult{
					urlID:    item.URLID,
					url:      item.NormalizedURL,
					host:     item.Host,
					depth:    item.Depth,
					fetchSeq: seq,
					result:   result,
					err:      fetchErr,
				}

				// Update URL status
				if fetchErr != nil {
					e.db.UpdateURLStatus(item.URLID, "errored")
					detailsJSON := fmt.Sprintf(`{"error":%q,"url":%q}`, fetchErr.Error(), item.NormalizedURL)
					e.db.InsertEvent(jobID, "fetch_error", &detailsJSON, &item.NormalizedURL)
				} else {
					e.db.UpdateURLStatus(item.URLID, "fetched")
				}

				select {
				case fetchResults <- fr:
				case <-ctx.Done():
					inFlight.Add(-1)
					return
				}
			}
		}()
	}

	// ---- Dispatcher: pulls from frontier and sends to fetchQueue ----
	// This runs on the main goroutine's ticker loop.
	// inFlight is incremented HERE (before sending to fetchQueue) and
	// decremented in the persister (after persist completes).
	// This eliminates the race between pop and tracking.

	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	var completionErr error
loop:
	for {
		// Drain frontier as much as possible
		for {
			item, ok := q.Pop()
			if !ok {
				break
			}
			inFlight.Add(1)
			select {
			case fetchQueue <- item:
			case <-ctx.Done():
				inFlight.Add(-1)
				completionErr = context.Cause(ctx)
				break loop
			}
		}

		// Check completion: nothing in queue and nothing in flight
		if q.Len() == 0 && inFlight.Load() == 0 {
			break loop
		}

		// Check for cancellation (includes persister fatal errors via WithCancelCause)
		select {
		case <-ctx.Done():
			completionErr = context.Cause(ctx)
			break loop
		case <-ticker.C:
			// Continue loop to check frontier again
		}
	}

	// Shutdown pipeline in order
	close(fetchQueue)
	fetcherWg.Wait()

	close(fetchResults)
	parserWg.Wait()

	close(persistQueue)
	persisterWg.Wait()

	// --- Post-crawl: HEAD-check discovered image assets ---
	if completionErr == nil {
		e.headCheckImageAssets(ctx, jobID)
	}

	// Update final counters
	e.db.UpdateJobCounters(jobID, int(pagesCrawled.Load()), int(urlsDiscovered.Load()), int(issuesFound.Load()))

	// Set final status
	if completionErr != nil {
		if ctx.Err() != nil {
			e.db.UpdateJobFinished(jobID, "cancelled", nil)
			return ctx.Err()
		}
		errMsg := completionErr.Error()
		e.db.UpdateJobFinished(jobID, "failed", &errMsg)
		return completionErr
	}

	e.db.UpdateJobFinished(jobID, "completed", nil)
	return nil
}

// processParseResult handles parsing, edge building, issue detection, and frontier expansion.
func (e *Engine) processParseResult(
	ctx context.Context,
	jobID string,
	fr fetchResult,
	q *frontier.Queue,
	pagesCrawled *atomic.Int64,
	urlsDiscovered *atomic.Int64,
	queryVariantsMu *sync.Mutex,
	queryVariants map[string]int,
) parseResult {
	pr := parseResult{
		fetchResult: fr,
		edges:       []crawl.DiscoveredEdge{},
		issues:      []issues.DetectedIssue{},
	}

	if fr.err != nil || fr.result == nil {
		return pr
	}

	// Rate limited detection
	if fr.result.StatusCode == 429 {
		pr.issues = append(pr.issues, issues.DetectedIssue{
			IssueType:   "rate_limited",
			Severity:    "info",
			Scope:       "page_local",
			DetailsJSON: fmt.Sprintf(`{"statusCode":429,"host":%q}`, fr.host),
		})
	}

	// Check if HTML
	ct := strings.ToLower(fr.result.ContentType)
	isHTML := strings.Contains(ct, "text/html")

	if !isHTML {
		return pr
	}

	// Parse HTML
	page, parseErr := parser.ParseHTML(fr.result.Body, fr.result.FinalURL, fr.result.ResponseHeaders)
	if parseErr != nil {
		detailsJSON := fmt.Sprintf(`{"error":%q,"url":%q}`, parseErr.Error(), fr.url)
		e.db.InsertEvent(jobID, "parse_error", &detailsJSON, &fr.url)
		return pr
	}
	pr.page = page

	// forceRenderPatterns: mark pages matching patterns as JS-suspect
	// so they get flagged for browser rendering in hybrid mode.
	if e.config.MatchesForceRender(fr.url) && !page.JSSuspect {
		page.JSSuspect = true
	}

	newCount := pagesCrawled.Add(1)
	if int(newCount) > e.config.MaxPages {
		// Past limit — don't expand edges from this page
		return pr
	}

	// Build edges
	pr.edges = crawl.BuildEdges(fr.urlID, fr.result.FinalURL, page, e.scopeChecker, "static")

	// Detect page-local issues
	thresholds := issues.Thresholds{
		TitleMaxLength:       e.config.TitleMaxLength,
		TitleMinLength:       e.config.TitleMinLength,
		DescriptionMaxLength: e.config.DescriptionMaxLength,
		DescriptionMinLength: e.config.DescriptionMinLength,
		ThinContentThreshold: e.config.ThinContentThreshold,
		DeepPageThreshold:    e.config.DeepPageThreshold,
	}
	pageCtx := issues.PageContext{
		StatusCode:           fr.result.StatusCode,
		RedirectHopCount:     len(fr.result.RedirectHops),
		RedirectLoopDetected: fr.result.RedirectLoopDetected,
		RedirectHopsExceeded: fr.result.RedirectHopsExceeded,
		TTFBMS:               fr.result.TTFBMS,
		ContentType:          fr.result.ContentType,
		Title:                page.Title,
		TitleLength:          page.TitleLength,
		MetaDescription:      page.MetaDescription,
		DescriptionLength:    page.DescriptionLength,
		MetaRobots:           page.MetaRobots,
		XRobotsTag:           page.XRobotsTag,
		CanonicalType:        page.CanonicalType,
		H1Count:              len(page.Headings.H1),
		OGTitle:              page.OpenGraph.Title,
		OGDescription:        page.OpenGraph.Description,
		OGImage:              page.OpenGraph.Image,
		JSONLDBlocks:         len(page.JSONLDBlocks),
		MalformedJSONLD:      hasMalformedJSONLD(page.JSONLDBlocks),
		JSONLDRaw:            marshalJSONLDBlocks(page.JSONLDBlocks),
		WordCount:            page.ExtractedWordCount,
		MainContentWordCount: page.MainContentWordCount,
		ImagesWithoutAlt:     countImagesWithoutAlt(page.Images),
		ImagesWithEmptyAlt:   countImagesWithEmptyAlt(page.Images),
		JSSuspect:             page.JSSuspect,
		ScriptCount:           page.ScriptCount,
		HasSPARoot:            page.HasSPARoot,
		TitleOutsideHead:      page.TitleOutsideHead,
		MetaRobotsOutsideHead: page.MetaRobotsOutsideHead,
	}
	pr.issues = issues.DetectPageLocalIssues(pageCtx, thresholds, fr.depth)

	// Collect discovered images for asset tracking
	for _, img := range page.Images {
		if img.Src == "" {
			continue
		}
		// Resolve relative URL against the page's final URL
		resolved := urlutil.ResolveReference(fr.result.FinalURL, img.Src)
		if resolved == "" {
			continue
		}
		imgNorm, normErr := urlutil.Normalize(resolved)
		if normErr != nil {
			continue
		}
		// Skip data: URLs
		if strings.HasPrefix(imgNorm, "data:") {
			continue
		}
		imgParsed, parseErr := url.Parse(imgNorm)
		if parseErr != nil {
			continue
		}
		imgHost := imgParsed.Hostname()
		imgInternal := false
		if e.scopeChecker != nil {
			imgInternal = e.scopeChecker.IsInScope(imgNorm)
		}
		pr.images = append(pr.images, discoveredImage{
			normalizedURL: imgNorm,
			host:          imgHost,
			isInternal:    imgInternal,
			sourceURLID:   fr.urlID,
		})
	}

	// Expand frontier with discovered in-scope links
	for _, edge := range pr.edges {
		if edge.RelationType != "link" {
			continue
		}
		if !edge.IsInternal {
			continue
		}

		normalized := edge.NormalizedTargetURL
		if normalized == "" {
			continue
		}

		// MaxDepth check
		newDepth := fr.depth + 1
		if newDepth > e.config.MaxDepth {
			continue
		}

		// MaxPages check
		if int(pagesCrawled.Load()) >= e.config.MaxPages {
			continue
		}

		// Crawl trap: repeated path segments
		if urlutil.HasRepeatedPathSegments(normalized) {
			continue
		}

		// Crawl trap: query variant limit
		parsed, parseErr := url.Parse(normalized)
		if parseErr != nil {
			continue
		}
		pathKey := parsed.Path
		if parsed.RawQuery != "" {
			queryVariantsMu.Lock()
			queryVariants[pathKey]++
			count := queryVariants[pathKey]
			queryVariantsMu.Unlock()
			if count > e.config.MaxQueryVariantsPerPath {
				detailsJSON := fmt.Sprintf(`{"path":%q,"queryVariants":%d,"limit":%d,"url":%q}`,
					pathKey, count, e.config.MaxQueryVariantsPerPath, normalized)
				e.db.InsertIssue(storage.IssueInput{
					JobID:       jobID,
					URLID:       nil,
					IssueType:   "crawl_trap_suspected",
					Severity:    "info",
					Scope:       "page_local",
					DetailsJSON: &detailsJSON,
				})
				continue
			}
		}

		targetHost := parsed.Hostname()
		urlID, upsertErr := e.db.UpsertURL(jobID, normalized, targetHost, "queued", true, "link")
		if upsertErr != nil {
			continue
		}

		urlsDiscovered.Add(1)

		q.Push(frontier.Item{
			URLID:         urlID,
			NormalizedURL: normalized,
			Host:          targetHost,
			Depth:         newDepth,
		})
	}

	return pr
}

// persistItem saves a single crawl result to the database inside a single transaction.
// headCheckImageAssets performs HEAD requests on all discovered image URLs
// and stores the results in the assets table. Caps at 1000 unique images.
func (e *Engine) headCheckImageAssets(ctx context.Context, jobID string) {
	// Query all distinct asset URLs from asset_references for this job
	rows, err := e.db.Query(
		`SELECT DISTINCT ar.asset_url_id, u.normalized_url
		 FROM asset_references ar
		 JOIN urls u ON u.id = ar.asset_url_id
		 WHERE ar.job_id = ?
		 LIMIT 1000`,
		jobID,
	)
	if err != nil {
		log.Printf("engine: failed to query image assets for HEAD checking: %v", err)
		return
	}
	defer rows.Close()

	type imageTarget struct {
		urlID int64
		url   string
	}
	var targets []imageTarget
	for rows.Next() {
		var t imageTarget
		if scanErr := rows.Scan(&t.urlID, &t.url); scanErr != nil {
			continue
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		log.Printf("engine: error iterating image assets: %v", err)
	}

	if len(targets) == 0 {
		return
	}

	log.Printf("engine: HEAD-checking %d discovered image assets", len(targets))

	// Use a small worker pool to avoid overwhelming hosts
	const headWorkers = 4
	work := make(chan imageTarget, len(targets))
	for _, t := range targets {
		work <- t
	}
	close(work)

	var wg sync.WaitGroup
	for range headWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range work {
				if ctx.Err() != nil {
					return
				}
				headResult, headErr := e.fetcher.Head(t.url)
				var contentType *string
				var statusCode *int
				var contentLength *int64
				if headErr == nil && headResult != nil {
					contentType = strPtr(headResult.ContentType)
					statusCode = intPtr(headResult.StatusCode)
					// Extract Content-Length from response headers
					if clStr := headResult.ResponseHeaders.Get("Content-Length"); clStr != "" {
						if cl, parseErr := strconv.ParseInt(clStr, 10, 64); parseErr == nil {
							contentLength = &cl
						}
					}
				}
				if _, insertErr := e.db.InsertAsset(storage.AssetInput{
					JobID:         jobID,
					URLID:         t.urlID,
					ContentType:   contentType,
					StatusCode:    statusCode,
					ContentLength: contentLength,
				}); insertErr != nil {
					// May fail on duplicate; that's fine
					continue
				}
			}
		}()
	}
	wg.Wait()
	log.Printf("engine: completed HEAD-checking image assets")
}

func (e *Engine) persistItem(ctx context.Context, jobID string, item persistItem) error {
	fr := item.fetchResult
	seq := item.fetchSeq

	if fr.result == nil && fr.err != nil {
		return nil
	}

	tx, err := e.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	// --- Resolve final URL ID (may upsert) ---
	var finalURLID *int64
	if fr.result != nil && fr.result.FinalURL != fr.url {
		parsed, parseErr := url.Parse(fr.result.FinalURL)
		if parseErr == nil {
			finalInScope := e.scopeChecker.IsInScope(fr.result.FinalURL)
			finalStatus := "fetched"
			if !finalInScope {
				finalStatus = "out_of_scope"
			}
			fid, upsertErr := txUpsertURL(tx, jobID, fr.result.FinalURL, parsed.Hostname(), finalStatus, finalInScope, "redirect")
			if upsertErr == nil {
				finalURLID = &fid
			}
		}
	}

	// --- Build fetch fields ---
	var fetchErr *string
	statusCode := 0
	var bodySize int64
	var contentType, contentEncoding string
	var ttfbMS int64
	redirectHopCount := 0
	var headersJSON string

	if fr.result != nil {
		statusCode = fr.result.StatusCode
		bodySize = fr.result.BodySize
		contentType = fr.result.ContentType
		contentEncoding = fr.result.ContentEncoding
		ttfbMS = fr.result.TTFBMS
		redirectHopCount = len(fr.result.RedirectHops)

		if h := fr.result.ResponseHeaders; h != nil {
			hBytes, _ := json.Marshal(h)
			headersJSON = string(hBytes)
		}
	}
	if fr.err != nil {
		s := fr.err.Error()
		fetchErr = &s
	}

	// --- Insert fetch ---
	result, insertErr := tx.ExecContext(ctx,
		`INSERT INTO fetches (job_id, fetch_seq, requested_url_id, final_url_id,
			status_code, redirect_hop_count, ttfb_ms, response_body_size,
			content_type, content_encoding, response_headers_json,
			http_method, fetch_kind, render_mode, render_params_json, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		jobID, seq, fr.urlID, finalURLID,
		statusCode, redirectHopCount, ttfbMS, bodySize,
		contentType, contentEncoding, headersJSON,
		"GET", "full", "static", nil, fetchErr,
	)
	if insertErr != nil {
		return fmt.Errorf("inserting fetch: %w", insertErr)
	}
	fetchID, _ := result.LastInsertId()

	// --- Insert redirect hops ---
	if fr.result != nil {
		for _, hop := range fr.result.RedirectHops {
			if _, hopErr := tx.ExecContext(ctx,
				`INSERT INTO redirect_hops (job_id, fetch_id, hop_index, status_code, from_url, to_url) VALUES (?, ?, ?, ?, ?, ?)`,
				jobID, fetchID, hop.HopIndex, hop.StatusCode, hop.FromURL, hop.ToURL,
			); hopErr != nil {
				return fmt.Errorf("inserting redirect hop: %w", hopErr)
			}
		}
	}

	ct := ""
	if fr.result != nil {
		ct = strings.ToLower(fr.result.ContentType)
	}
	isHTML := strings.Contains(ct, "text/html")

	// --- Non-HTML: record as asset ---
	if !isHTML && fr.result != nil && fr.err == nil {
		if _, assetErr := tx.ExecContext(ctx,
			`INSERT INTO assets (job_id, url_id, content_type, status_code, content_length) VALUES (?, ?, ?, ?, ?)`,
			jobID, fr.urlID, fr.result.ContentType, fr.result.StatusCode, fr.result.BodySize,
		); assetErr != nil {
			return fmt.Errorf("inserting asset: %w", assetErr)
		}
		return tx.Commit()
	}

	// --- HTML: insert page record ---
	if isHTML && item.page != nil {
		if pageErr := txInsertPage(ctx, tx, jobID, fr.urlID, fetchID, fr.depth, item.page); pageErr != nil {
			return fmt.Errorf("inserting page: %w", pageErr)
		}
	}

	// --- Insert edges ---
	for _, edge := range item.edges {
		parsed, parseErr := url.Parse(edge.NormalizedTargetURL)
		if parseErr != nil {
			continue
		}
		targetHost := parsed.Hostname()
		targetURLID, upsertErr := txUpsertURL(tx, jobID, edge.NormalizedTargetURL, targetHost, "discovered", edge.IsInternal, "link")
		if upsertErr != nil {
			continue
		}

		var anchorText *string
		if edge.AnchorText != "" {
			anchorText = &edge.AnchorText
		}
		var relFlags *string
		if edge.RelFlagsJSON != "" {
			relFlags = &edge.RelFlagsJSON
		}

		boolToInt := 0
		if edge.IsInternal {
			boolToInt = 1
		}

		// HEAD request for out-of-scope canonical/hreflang targets
		var targetStatusCode *int
		if !edge.IsInternal && (edge.RelationType == "canonical" || edge.RelationType == "hreflang") {
			headResult, headErr := e.fetcher.Head(edge.NormalizedTargetURL)
			if headErr == nil && headResult != nil {
				targetStatusCode = &headResult.StatusCode
			}
		}

		if _, edgeErr := tx.ExecContext(ctx,
			`INSERT INTO edges (job_id, source_url_id, normalized_target_url_id,
				source_kind, relation_type, rel_flags_json, discovery_mode,
				anchor_text, is_internal, declared_target_url, target_status_code)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			jobID, edge.SourceURLID, targetURLID,
			edge.SourceKind, edge.RelationType, relFlags, edge.DiscoveryMode,
			anchorText, boolToInt, edge.DeclaredTargetURL, targetStatusCode,
		); edgeErr != nil {
			return fmt.Errorf("inserting edge: %w", edgeErr)
		}
	}

	// --- Insert issues ---
	for _, issue := range item.issues {
		details := issue.DetailsJSON
		if _, issueErr := tx.ExecContext(ctx,
			`INSERT INTO issues (job_id, url_id, issue_type, severity, scope, details_json) VALUES (?, ?, ?, ?, ?, ?)`,
			jobID, &fr.urlID, issue.IssueType, issue.Severity, issue.Scope, &details,
		); issueErr != nil {
			return fmt.Errorf("inserting issue: %w", issueErr)
		}
	}

	// --- Insert image asset references ---
	for _, img := range item.images {
		imgURLID, upsertErr := txUpsertURL(tx, jobID, img.normalizedURL, img.host, "discovered", img.isInternal, "asset")
		if upsertErr != nil {
			continue
		}
		if _, refErr := tx.ExecContext(ctx,
			`INSERT INTO asset_references (job_id, asset_url_id, source_page_url_id, reference_type)
			 VALUES (?, ?, ?, ?)`,
			jobID, imgURLID, img.sourceURLID, "img_src",
		); refErr != nil {
			// Duplicate references are possible; ignore unique constraint errors
			continue
		}
	}

	return tx.Commit()
}

// txUpsertURL upserts a URL within a transaction and returns its ID.
func txUpsertURL(tx *sql.Tx, jobID, normalizedURL, host, status string, isInternal bool, discoveredVia string) (int64, error) {
	isInternalInt := 0
	if isInternal {
		isInternalInt = 1
	}
	_, err := tx.Exec(
		`INSERT OR IGNORE INTO urls (job_id, normalized_url, host, status, is_internal, discovered_via)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		jobID, normalizedURL, host, status, isInternalInt, discoveredVia,
	)
	if err != nil {
		return 0, fmt.Errorf("upserting URL %q: %w", normalizedURL, err)
	}

	var id int64
	err = tx.QueryRow(
		`SELECT id FROM urls WHERE job_id = ? AND normalized_url = ?`,
		jobID, normalizedURL,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("fetching ID for URL %q: %w", normalizedURL, err)
	}
	return id, nil
}

// txInsertPage creates a page record within a transaction.
func txInsertPage(ctx context.Context, tx *sql.Tx, jobID string, urlID, fetchID int64, depth int, page *parser.ParseResult) error {
	title := strPtr(page.Title)
	titleLen := intPtr(page.TitleLength)
	metaDesc := strPtr(page.MetaDescription)
	metaDescLen := intPtr(page.DescriptionLength)
	metaRobots := strPtr(page.MetaRobots)
	xRobots := strPtr(page.XRobotsTag)
	canonical := strPtr(page.CanonicalResolved)

	var canonicalIsSelf *int
	if page.CanonicalType == "self" {
		v := 1
		canonicalIsSelf = &v
	} else if page.CanonicalType == "cross" {
		v := 0
		canonicalIsSelf = &v
	}

	var relNext, relPrev *string
	if page.RelNext != nil {
		relNext = &page.RelNext.Resolved
	}
	if page.RelPrev != nil {
		relPrev = &page.RelPrev.Resolved
	}

	hreflangJSON := jsonStrPtr(page.Hreflangs)
	h1JSON := jsonStrPtr(page.Headings.H1)
	h2JSON := jsonStrPtr(page.Headings.H2)
	h3JSON := jsonStrPtr(page.Headings.H3)
	h4JSON := jsonStrPtr(page.Headings.H4)
	h5JSON := jsonStrPtr(page.Headings.H5)
	h6JSON := jsonStrPtr(page.Headings.H6)

	ogTitle := strPtr(page.OpenGraph.Title)
	ogDesc := strPtr(page.OpenGraph.Description)
	ogImage := strPtr(page.OpenGraph.Image)
	ogURL := strPtr(page.OpenGraph.URL)
	ogType := strPtr(page.OpenGraph.Type)

	twitterCard := strPtr(page.TwitterCard.Card)
	twitterTitle := strPtr(page.TwitterCard.Title)
	twitterDesc := strPtr(page.TwitterCard.Description)
	twitterImage := strPtr(page.TwitterCard.Image)

	var jsonldRaw *string
	if len(page.JSONLDBlocks) > 0 {
		raw, _ := json.Marshal(page.JSONLDBlocks)
		s := string(raw)
		jsonldRaw = &s
	}
	jsonldTypes := jsonStrPtr(page.JSONLDTypes)
	imagesJSON := jsonStrPtr(page.Images)
	wordCount := intPtr(page.ExtractedWordCount)
	mainWC := intPtr(page.MainContentWordCount)
	contentHash := strPtr(page.ContentHash)

	jsSuspect := 0
	if page.JSSuspect {
		jsSuspect = 1
	}

	_, err := tx.ExecContext(ctx,
		`INSERT INTO pages (job_id, url_id, fetch_id, depth,
			title, title_length, meta_description, meta_description_length,
			meta_robots, x_robots_tag, indexability_state,
			canonical_url, canonical_is_self, rel_next_url, rel_prev_url,
			hreflang_json,
			h1_json, h2_json, h3_json, h4_json, h5_json, h6_json,
			og_title, og_description, og_image, og_url, og_type,
			twitter_card, twitter_title, twitter_description, twitter_image,
			jsonld_raw, jsonld_types_json,
			images_json, word_count, main_content_word_count,
			content_hash, js_suspect)
		 VALUES (?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?,
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?)`,
		jobID, urlID, fetchID, depth,
		title, titleLen, metaDesc, metaDescLen,
		metaRobots, xRobots, page.IndexabilityState,
		canonical, canonicalIsSelf, relNext, relPrev,
		hreflangJSON,
		h1JSON, h2JSON, h3JSON, h4JSON, h5JSON, h6JSON,
		ogTitle, ogDesc, ogImage, ogURL, ogType,
		twitterCard, twitterTitle, twitterDesc, twitterImage,
		jsonldRaw, jsonldTypes,
		imagesJSON, wordCount, mainWC,
		contentHash, jsSuspect,
	)
	return err
}

// failJob marks a job as failed with the given error.
func (e *Engine) failJob(jobID string, err error) error {
	errMsg := err.Error()
	e.db.UpdateJobFinished(jobID, "failed", &errMsg)
	return err
}

// Helper functions

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	return &i
}

func jsonStrPtr(v any) *string {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(data)
	return &s
}

func marshalJSONLDBlocks(blocks []parser.JSONLDBlock) string {
	if len(blocks) == 0 {
		return ""
	}
	raw, err := json.Marshal(blocks)
	if err != nil {
		return ""
	}
	return string(raw)
}

func hasMalformedJSONLD(blocks []parser.JSONLDBlock) bool {
	for _, b := range blocks {
		if b.Malformed {
			return true
		}
	}
	return false
}

func countImagesWithoutAlt(images []parser.DiscoveredImage) int {
	count := 0
	for _, img := range images {
		if img.AltMissing {
			count++
		}
	}
	return count
}

func countImagesWithEmptyAlt(images []parser.DiscoveredImage) int {
	count := 0
	for _, img := range images {
		if img.AltEmpty {
			count++
		}
	}
	return count
}

