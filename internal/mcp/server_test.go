package mcp

import (
	"context"
	"encoding/json"
	"testing"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

func TestAllToolsRegistered(t *testing.T) {
	s := NewServer(ServerConfig{})

	expectedTools := []string{
		"crawl_site", "crawl_status", "cancel_crawl",
		"get_crawl_summary", "get_crawl_results", "get_link_graph",
		"analyze_url", "check_redirects", "check_robots_txt", "parse_sitemap",
	}

	// Use the initialize flow to list tools
	initReq := gomcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = "2024-11-05"
	initReq.Params.ClientInfo = gomcp.Implementation{Name: "test", Version: "0.1"}

	ctx := context.Background()

	// Call ListTools via the MCP server
	listReq := gomcp.ListToolsRequest{}
	listReqJSON, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  listReq.Params,
	})

	// Use HandleMessage to get tools list
	resp := s.mcpServer.HandleMessage(ctx, listReqJSON)

	// Parse response
	respJSON, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshalling response: %v", err)
	}

	var result struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respJSON, &result); err != nil {
		t.Fatalf("unmarshalling response: %v", err)
	}

	registered := make(map[string]bool)
	for _, tool := range result.Result.Tools {
		registered[tool.Name] = true
	}

	for _, name := range expectedTools {
		if !registered[name] {
			t.Errorf("tool %q not registered", name)
		}
	}

	if len(result.Result.Tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(result.Result.Tools))
	}
}

func TestNewServer_SetsNameAndVersion(t *testing.T) {
	s := NewServer(ServerConfig{})
	if s.mcpServer == nil {
		t.Fatal("mcpServer is nil")
	}
}

func TestToolAnnotations(t *testing.T) {
	s := NewServer(ServerConfig{})

	ctx := context.Background()
	listReqJSON, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})

	resp := s.mcpServer.HandleMessage(ctx, listReqJSON)
	respJSON, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshalling response: %v", err)
	}

	var result struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				Annotations struct {
					ReadOnlyHint  *bool `json:"readOnlyHint"`
					OpenWorldHint *bool `json:"openWorldHint"`
				} `json:"annotations"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respJSON, &result); err != nil {
		t.Fatalf("unmarshalling response: %v", err)
	}

	expected := map[string][2]bool{
		"crawl_site":        {false, true},
		"crawl_status":      {true, false},
		"cancel_crawl":      {false, false},
		"get_crawl_summary": {true, false},
		"get_crawl_results": {true, false},
		"get_link_graph":    {true, false},
		"analyze_url":       {false, true},
		"check_redirects":   {true, true},
		"check_robots_txt":  {true, true},
		"parse_sitemap":     {true, true},
	}

	toolMap := make(map[string]struct {
		ReadOnlyHint  *bool
		OpenWorldHint *bool
	})
	for _, tool := range result.Result.Tools {
		toolMap[tool.Name] = struct {
			ReadOnlyHint  *bool
			OpenWorldHint *bool
		}{tool.Annotations.ReadOnlyHint, tool.Annotations.OpenWorldHint}
	}

	for name, exp := range expected {
		tool, ok := toolMap[name]
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		if tool.ReadOnlyHint == nil {
			t.Errorf("tool %q: readOnlyHint is nil, want %v", name, exp[0])
		} else if *tool.ReadOnlyHint != exp[0] {
			t.Errorf("tool %q: readOnlyHint = %v, want %v", name, *tool.ReadOnlyHint, exp[0])
		}
		if tool.OpenWorldHint == nil {
			t.Errorf("tool %q: openWorldHint is nil, want %v", name, exp[1])
		} else if *tool.OpenWorldHint != exp[1] {
			t.Errorf("tool %q: openWorldHint = %v, want %v", name, *tool.OpenWorldHint, exp[1])
		}
	}
}
