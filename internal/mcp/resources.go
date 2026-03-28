package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

// registerResources adds all resource and resource template handlers.
func (s *Server) registerResources() {
	// Static resource: job list
	s.mcpServer.AddResource(gomcp.NewResource(
		"seo-crawler://jobs",
		"List of all crawl jobs",
		gomcp.WithMIMEType("application/json"),
	), s.handleJobListResource)

	// Dynamic resource templates
	s.mcpServer.AddResourceTemplate(gomcp.NewResourceTemplate(
		"seo-crawler://jobs/{jobId}",
		"Crawl job details",
		gomcp.WithTemplateMIMEType("application/json"),
	), s.handleJobDetailResource)

	s.mcpServer.AddResourceTemplate(gomcp.NewResourceTemplate(
		"seo-crawler://jobs/{jobId}/summary",
		"Crawl summary snapshot",
		gomcp.WithTemplateMIMEType("application/json"),
	), s.handleJobSummaryResource)

	s.mcpServer.AddResourceTemplate(gomcp.NewResourceTemplate(
		"seo-crawler://jobs/{jobId}/events",
		"Crawl event log",
		gomcp.WithTemplateMIMEType("application/json"),
	), s.handleJobEventsResource)
}

// handleJobListResource returns all crawl jobs as JSON.
func (s *Server) handleJobListResource(
	ctx context.Context,
	req gomcp.ReadResourceRequest,
) ([]gomcp.ResourceContents, error) {
	jobs, err := s.db.ListJobs()
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}

	data, err := json.Marshal(jobs)
	if err != nil {
		return nil, fmt.Errorf("marshalling jobs: %w", err)
	}

	return []gomcp.ResourceContents{
		gomcp.TextResourceContents{
			URI:      "seo-crawler://jobs",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// extractJobID extracts the job ID from a resource URI.
// Expected formats: seo-crawler://jobs/{jobId}[/suffix]
func extractJobID(uri string) (string, error) {
	const prefix = "seo-crawler://jobs/"
	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("invalid resource URI %q", uri)
	}
	rest := strings.TrimPrefix(uri, prefix)
	// Take everything up to the next slash (or end)
	parts := strings.SplitN(rest, "/", 2)
	if parts[0] == "" {
		return "", fmt.Errorf("missing job ID in URI %q", uri)
	}
	return parts[0], nil
}

// jobDetailPayload is the JSON structure for the job detail resource.
type jobDetailPayload struct {
	Job            any            `json:"job"`
	URLsByStatus   map[string]int `json:"urlsByStatus"`
	IssuesByType   map[string]int `json:"issuesByType"`
}

// handleJobDetailResource returns full job detail with counters.
func (s *Server) handleJobDetailResource(
	ctx context.Context,
	req gomcp.ReadResourceRequest,
) ([]gomcp.ResourceContents, error) {
	jobID, err := extractJobID(req.Params.URI)
	if err != nil {
		return nil, err
	}

	job, err := s.db.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("getting job %q: %w", jobID, err)
	}

	urlsByStatus, err := s.db.CountURLsByStatus(jobID)
	if err != nil {
		return nil, fmt.Errorf("counting URLs for job %q: %w", jobID, err)
	}

	issuesByType, err := s.db.CountIssuesByType(jobID)
	if err != nil {
		return nil, fmt.Errorf("counting issues for job %q: %w", jobID, err)
	}

	payload := jobDetailPayload{
		Job:          job,
		URLsByStatus: urlsByStatus,
		IssuesByType: issuesByType,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshalling job detail: %w", err)
	}

	return []gomcp.ResourceContents{
		gomcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// handleJobSummaryResource returns the crawl summary snapshot.
func (s *Server) handleJobSummaryResource(
	ctx context.Context,
	req gomcp.ReadResourceRequest,
) ([]gomcp.ResourceContents, error) {
	jobID, err := extractJobID(req.Params.URI)
	if err != nil {
		return nil, err
	}

	summary, err := s.db.GetCrawlSummary(jobID)
	if err != nil {
		return nil, fmt.Errorf("getting summary for job %q: %w", jobID, err)
	}

	data, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("marshalling summary: %w", err)
	}

	return []gomcp.ResourceContents{
		gomcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// handleJobEventsResource returns the crawl event log.
func (s *Server) handleJobEventsResource(
	ctx context.Context,
	req gomcp.ReadResourceRequest,
) ([]gomcp.ResourceContents, error) {
	jobID, err := extractJobID(req.Params.URI)
	if err != nil {
		return nil, err
	}

	events, err := s.db.GetEventsByJob(jobID, 1000)
	if err != nil {
		return nil, fmt.Errorf("getting events for job %q: %w", jobID, err)
	}

	data, err := json.Marshal(events)
	if err != nil {
		return nil, fmt.Errorf("marshalling events: %w", err)
	}

	return []gomcp.ResourceContents{
		gomcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
