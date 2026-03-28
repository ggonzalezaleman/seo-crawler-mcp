package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

// crawlSiteArgs holds parsed arguments for crawl_site.
type crawlSiteArgs struct {
	URL           string   `json:"url"`
	URLs          []string `json:"urls,omitempty"`
	ScopeMode     string   `json:"scopeMode,omitempty"`
	AllowedHosts  []string `json:"allowedHosts,omitempty"`
	MaxPages      int      `json:"maxPages,omitempty"`
	MaxDepth      int      `json:"maxDepth,omitempty"`
	RenderMode    string   `json:"renderMode,omitempty"`
	RespectRobots *bool    `json:"respectRobots,omitempty"`
	DryRun        bool     `json:"dryRun,omitempty"`
}

// crawlSiteResult is returned from crawl_site.
type crawlSiteResult struct {
	JobID        string `json:"jobId"`
	Status       string `json:"status"`
	ResourceLink string `json:"resourceLink"`
}

// crawlStatusResult is returned from crawl_status.
type crawlStatusResult struct {
	JobID          string         `json:"jobId"`
	Status         string         `json:"status"`
	Type           string         `json:"type"`
	CreatedAt      string         `json:"createdAt"`
	StartedAt      string         `json:"startedAt,omitempty"`
	FinishedAt     string         `json:"finishedAt,omitempty"`
	Error          string         `json:"error,omitempty"`
	PagesCrawled   int            `json:"pagesCrawled"`
	URLsDiscovered int            `json:"urlsDiscovered"`
	IssuesFound    int            `json:"issuesFound"`
	URLsByStatus   map[string]int `json:"urlsByStatus,omitempty"`
	IssuesByType   map[string]int `json:"issuesByType,omitempty"`
}

func (s *Server) handleCrawlSite(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	args := req.GetArguments()

	rawURL, ok := args["url"].(string)
	if !ok || rawURL == "" {
		return gomcp.NewToolResultError("parameter \"url\" is required"), nil
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return gomcp.NewToolResultError(fmt.Sprintf("invalid URL %q: must be http or https", rawURL)), nil
	}

	// Parse optional scope mode and validate
	scopeMode := "registrable_domain"
	if sm, ok := args["scopeMode"].(string); ok && sm != "" {
		switch sm {
		case "registrable_domain", "exact_host", "allowlist":
			scopeMode = sm
		default:
			return gomcp.NewToolResultError(fmt.Sprintf("invalid scopeMode %q", sm)), nil
		}
	}

	// Parse optional numeric params
	maxPages := 10000
	if mp, ok := args["maxPages"].(float64); ok && mp > 0 {
		maxPages = int(mp)
	}
	const maxPagesLimit = 100000
	if maxPages > maxPagesLimit {
		maxPages = maxPagesLimit
	}

	maxDepth := 50
	if md, ok := args["maxDepth"].(float64); ok && md > 0 {
		maxDepth = int(md)
	}

	renderMode := "static"
	if rm, ok := args["renderMode"].(string); ok && rm != "" {
		switch rm {
		case "static", "browser", "hybrid":
			renderMode = rm
		default:
			return gomcp.NewToolResultError(fmt.Sprintf("invalid renderMode %q", rm)), nil
		}
	}

	respectRobots := true
	if rr, ok := args["respectRobots"].(bool); ok {
		respectRobots = rr
	}

	dryRun := false
	if dr, ok := args["dryRun"].(bool); ok {
		dryRun = dr
	}

	// Collect additional URLs
	var additionalURLs []string
	if rawURLs, ok := args["urls"].([]any); ok {
		for _, u := range rawURLs {
			if us, ok := u.(string); ok {
				additionalURLs = append(additionalURLs, us)
			}
		}
	}

	var allowedHosts []string
	if rawHosts, ok := args["allowedHosts"].([]any); ok {
		for _, h := range rawHosts {
			if hs, ok := h.(string); ok {
				allowedHosts = append(allowedHosts, hs)
			}
		}
	}

	if s.db == nil {
		return gomcp.NewToolResultError("server not configured: database unavailable"), nil
	}

	// Job guard: check concurrent crawl limit
	maxConcurrent := 3
	if s.config != nil {
		maxConcurrent = s.config.MaxConcurrentCrawls
	}
	activeCount, err := s.db.CountActiveJobs("crawl")
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("checking active jobs: %v", err)), nil
	}
	if activeCount >= maxConcurrent {
		return gomcp.NewToolResultError(fmt.Sprintf("concurrent crawl limit reached (%d/%d active)", activeCount, maxConcurrent)), nil
	}

	// Build config JSON
	crawlConfig := map[string]any{
		"scopeMode":     scopeMode,
		"allowedHosts":  allowedHosts,
		"maxPages":      maxPages,
		"maxDepth":      maxDepth,
		"renderMode":    renderMode,
		"respectRobots": respectRobots,
		"dryRun":        dryRun,
	}
	configJSON, err := json.Marshal(crawlConfig)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("marshalling config: %v", err)), nil
	}

	// Build seed URLs list
	seedURLs := []string{rawURL}
	seedURLs = append(seedURLs, additionalURLs...)
	seedJSON, err := json.Marshal(seedURLs)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("marshalling seed URLs: %v", err)), nil
	}

	job, err := s.db.CreateJob("crawl", string(configJSON), string(seedJSON))
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("creating job: %v", err)), nil
	}

	// Start crawl in background (non-dryRun).
	// NOTE: We do NOT mutate s.config here — that would be a data race.
	// Per-job config is already stored in the job's config_json field above.
	// The engine reads config from the job record when starting a crawl.
	if !dryRun && s.engine != nil {
		go func() {
			_ = s.engine.RunCrawl(context.Background(), job.ID)
		}()
	}

	result := crawlSiteResult{
		JobID:        job.ID,
		Status:       job.Status,
		ResourceLink: fmt.Sprintf("seo-crawler://jobs/%s", job.ID),
	}

	return gomcp.NewToolResultJSON(result)
}

func (s *Server) handleCrawlStatus(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	args := req.GetArguments()

	jobID, ok := args["jobId"].(string)
	if !ok || jobID == "" {
		return gomcp.NewToolResultError("parameter \"jobId\" is required"), nil
	}

	if s.db == nil {
		return gomcp.NewToolResultError("server not configured: database unavailable"), nil
	}

	job, err := s.db.GetJob(jobID)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("job %q: %v", jobID, err)), nil
	}

	result := crawlStatusResult{
		JobID:          job.ID,
		Status:         job.Status,
		Type:           job.Type,
		CreatedAt:      job.CreatedAt,
		PagesCrawled:   job.PagesCrawled,
		URLsDiscovered: job.URLsDiscovered,
		IssuesFound:    job.IssuesFound,
	}

	if job.StartedAt.Valid {
		result.StartedAt = job.StartedAt.String
	}
	if job.FinishedAt.Valid {
		result.FinishedAt = job.FinishedAt.String
	}
	if job.Error.Valid {
		result.Error = job.Error.String
	}

	// URL counts by status
	urlCounts, err := s.db.CountURLsByStatus(jobID)
	if err == nil {
		result.URLsByStatus = urlCounts
	}

	// Issue counts by type
	issueCounts, err := s.db.CountIssuesByType(jobID)
	if err == nil {
		result.IssuesByType = issueCounts
	}

	return gomcp.NewToolResultJSON(result)
}

func (s *Server) handleCancelCrawl(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	args := req.GetArguments()

	jobID, ok := args["jobId"].(string)
	if !ok || jobID == "" {
		return gomcp.NewToolResultError("parameter \"jobId\" is required"), nil
	}

	if s.db == nil {
		return gomcp.NewToolResultError("server not configured: database unavailable"), nil
	}

	job, err := s.db.GetJob(jobID)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("job %q: %v", jobID, err)), nil
	}

	// Only running/queued jobs can be cancelled
	if job.Status != "running" && job.Status != "queued" {
		return gomcp.NewToolResultError(fmt.Sprintf("job %q has status %q, only running or queued jobs can be cancelled", jobID, job.Status)), nil
	}

	if err := s.db.UpdateJobStatus(jobID, "cancelling"); err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("cancelling job %q: %v", jobID, err)), nil
	}

	result := map[string]string{
		"jobId":  jobID,
		"status": "cancelling",
	}
	return gomcp.NewToolResultJSON(result)
}
