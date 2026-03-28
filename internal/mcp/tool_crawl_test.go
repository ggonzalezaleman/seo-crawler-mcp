package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/config"
	"github.com/ggonzalezaleman/seo-crawler-mcp/internal/storage"
	gomcp "github.com/mark3labs/mcp-go/mcp"
)

func setupTestDB(t *testing.T) *storage.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func callTool(t *testing.T, s *Server, args map[string]any) *gomcp.CallToolResult {
	t.Helper()
	req := gomcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := s.handleCrawlSite(context.Background(), req)
	if err != nil {
		t.Fatalf("handleCrawlSite returned error: %v", err)
	}
	return result
}

func TestCrawlSite_CreatesJob(t *testing.T) {
	db := setupTestDB(t)
	cfg := config.DefaultConfig()

	s := NewServer(ServerConfig{
		DB:     db,
		Config: &cfg,
	})

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"url": "https://example.com",
	}

	result, err := s.handleCrawlSite(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}

	// Parse result to get job ID
	var res crawlSiteResult
	for _, content := range result.Content {
		if tc, ok := content.(gomcp.TextContent); ok {
			if err := json.Unmarshal([]byte(tc.Text), &res); err != nil {
				t.Fatalf("parsing result: %v", err)
			}
		}
	}

	if res.JobID == "" {
		t.Fatal("expected non-empty job ID")
	}
	if res.Status != "queued" {
		t.Errorf("expected status %q, got %q", "queued", res.Status)
	}
	if res.ResourceLink == "" {
		t.Error("expected non-empty resource link")
	}

	// Verify job exists in DB
	job, err := db.GetJob(res.JobID)
	if err != nil {
		t.Fatalf("getting job from DB: %v", err)
	}
	if job.Type != "crawl" {
		t.Errorf("expected job type %q, got %q", "crawl", job.Type)
	}
}

func TestCrawlSite_RequiresURL(t *testing.T) {
	db := setupTestDB(t)
	s := NewServer(ServerConfig{DB: db})

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := s.handleCrawlSite(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for missing URL")
	}
}

func TestCrawlSite_InvalidURL(t *testing.T) {
	db := setupTestDB(t)
	s := NewServer(ServerConfig{DB: db})

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"url": "not-a-url",
	}

	result, err := s.handleCrawlSite(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid URL")
	}
}

func TestCrawlSite_InvalidScopeMode(t *testing.T) {
	db := setupTestDB(t)
	s := NewServer(ServerConfig{DB: db})

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"url":       "https://example.com",
		"scopeMode": "invalid_mode",
	}

	result, err := s.handleCrawlSite(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error result for invalid scope mode")
	}
}

func TestCrawlSite_JobGuard(t *testing.T) {
	db := setupTestDB(t)
	cfg := config.DefaultConfig()
	cfg.MaxConcurrentCrawls = 1

	s := NewServer(ServerConfig{
		DB:     db,
		Config: &cfg,
	})

	// Create first job (will be queued)
	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"url": "https://example.com",
	}
	result, err := s.handleCrawlSite(context.Background(), req)
	if err != nil {
		t.Fatalf("first crawl: %v", err)
	}
	if result.IsError {
		t.Fatalf("first crawl should succeed")
	}

	// Second job should be blocked
	req2 := gomcp.CallToolRequest{}
	req2.Params.Arguments = map[string]any{
		"url": "https://other.com",
	}
	result2, err := s.handleCrawlSite(context.Background(), req2)
	if err != nil {
		t.Fatalf("second crawl: %v", err)
	}

	if !result2.IsError {
		t.Error("expected error: concurrent crawl limit should be reached")
	}
}

func TestCrawlStatus_ReturnsCounters(t *testing.T) {
	db := setupTestDB(t)
	cfg := config.DefaultConfig()

	s := NewServer(ServerConfig{
		DB:     db,
		Config: &cfg,
	})

	// Create a job directly
	job, err := db.CreateJob("crawl", `{}`, `["https://example.com"]`)
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}

	// Update counters
	if err := db.UpdateJobCounters(job.ID, 42, 100, 5); err != nil {
		t.Fatalf("updating counters: %v", err)
	}

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"jobId": job.ID,
	}

	result, err := s.handleCrawlStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	var status crawlStatusResult
	for _, content := range result.Content {
		if tc, ok := content.(gomcp.TextContent); ok {
			if err := json.Unmarshal([]byte(tc.Text), &status); err != nil {
				t.Fatalf("parsing status: %v", err)
			}
		}
	}

	if status.PagesCrawled != 42 {
		t.Errorf("expected 42 pages crawled, got %d", status.PagesCrawled)
	}
	if status.URLsDiscovered != 100 {
		t.Errorf("expected 100 URLs discovered, got %d", status.URLsDiscovered)
	}
	if status.IssuesFound != 5 {
		t.Errorf("expected 5 issues found, got %d", status.IssuesFound)
	}
}

func TestCrawlStatus_JobNotFound(t *testing.T) {
	db := setupTestDB(t)
	s := NewServer(ServerConfig{DB: db})

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"jobId": "nonexistent-id",
	}

	result, err := s.handleCrawlStatus(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error for nonexistent job")
	}
}

func TestCancelCrawl_TransitionsStatus(t *testing.T) {
	db := setupTestDB(t)
	s := NewServer(ServerConfig{DB: db})

	// Create and start a job
	job, err := db.CreateJob("crawl", `{}`, `["https://example.com"]`)
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}
	if err := db.UpdateJobStarted(job.ID); err != nil {
		t.Fatalf("starting job: %v", err)
	}

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"jobId": job.ID,
	}

	result, err := s.handleCancelCrawl(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	// Verify status in DB
	updated, err := db.GetJob(job.ID)
	if err != nil {
		t.Fatalf("getting updated job: %v", err)
	}
	if updated.Status != "cancelling" {
		t.Errorf("expected status %q, got %q", "cancelling", updated.Status)
	}
}

func TestCancelCrawl_RejectsCompletedJob(t *testing.T) {
	db := setupTestDB(t)
	s := NewServer(ServerConfig{DB: db})

	job, err := db.CreateJob("crawl", `{}`, `["https://example.com"]`)
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}
	if err := db.UpdateJobFinished(job.ID, "completed", nil); err != nil {
		t.Fatalf("finishing job: %v", err)
	}

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"jobId": job.ID,
	}

	result, err := s.handleCancelCrawl(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error: cannot cancel completed job")
	}
}

func TestCrawlSite_NilDB(t *testing.T) {
	s := NewServer(ServerConfig{})

	req := gomcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"url": "https://example.com",
	}

	result, err := s.handleCrawlSite(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.IsError {
		t.Error("expected error when DB is nil")
	}
}


