package mcp

import (
	"context"
	"fmt"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) handleGetCrawlSummary(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	args := req.GetArguments()

	jobID, ok := args["jobId"].(string)
	if !ok || jobID == "" {
		return gomcp.NewToolResultError("parameter %q is required"), nil
	}

	if s.db == nil {
		return gomcp.NewToolResultError("server not configured: database unavailable"), nil
	}

	summary, err := s.db.GetCrawlSummary(jobID)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("getting summary for job %q: %v", jobID, err)), nil
	}

	return gomcp.NewToolResultJSON(summary)
}

func (s *Server) handleGetCrawlResults(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	args := req.GetArguments()

	jobID, ok := args["jobId"].(string)
	if !ok || jobID == "" {
		return gomcp.NewToolResultError("parameter %q is required"), nil
	}

	if s.db == nil {
		return gomcp.NewToolResultError("server not configured: database unavailable"), nil
	}

	limit := 100
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	var cursor string
	if c, ok := args["cursor"].(string); ok {
		cursor = c
	}

	issues, err := s.db.GetIssuesByJob(jobID, limit, cursor)
	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("querying results for job %q: %v", jobID, err)), nil
	}

	return gomcp.NewToolResultJSON(issues)
}

func (s *Server) handleGetLinkGraph(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	args := req.GetArguments()

	jobID, ok := args["jobId"].(string)
	if !ok || jobID == "" {
		return gomcp.NewToolResultError("parameter %q is required"), nil
	}

	if s.db == nil {
		return gomcp.NewToolResultError("server not configured: database unavailable"), nil
	}

	limit := 100
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	var cursor string
	if c, ok := args["cursor"].(string); ok {
		cursor = c
	}

	var urlID int64
	if uid, ok := args["urlId"].(float64); ok {
		urlID = int64(uid)
	}

	direction := "outbound"
	if d, ok := args["direction"].(string); ok && d != "" {
		direction = d
	}

	var edges any
	var err error
	if urlID > 0 {
		if direction == "inbound" {
			edges, err = s.db.GetEdgesByTarget(jobID, urlID, limit, cursor)
		} else {
			edges, err = s.db.GetEdgesBySource(jobID, urlID, limit, cursor)
		}
	} else {
		edges, err = s.db.GetEdgesBySource(jobID, 0, limit, cursor)
	}

	if err != nil {
		return gomcp.NewToolResultError(fmt.Sprintf("querying link graph for job %q: %v", jobID, err)), nil
	}

	return gomcp.NewToolResultJSON(edges)
}
