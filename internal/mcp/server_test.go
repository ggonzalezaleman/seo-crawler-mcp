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
