package mcp

import (
	gomcp "github.com/mark3labs/mcp-go/mcp"
)

// Tool definitions for all 10 MCP tools.
var (
	crawlSiteTool = gomcp.NewTool("crawl_site",
		gomcp.WithDescription("Start a new site crawl. Returns a job ID for tracking progress."),
		gomcp.WithString("url", gomcp.Required(), gomcp.Description("Seed URL to begin crawling")),
		gomcp.WithArray("urls", gomcp.Description("Additional seed URLs"), gomcp.WithStringItems()),
		gomcp.WithString("scopeMode", gomcp.Description("Crawl scope boundary"), gomcp.Enum("registrable_domain", "exact_host", "allowlist")),
		gomcp.WithArray("allowedHosts", gomcp.Description("Hosts to allow when scopeMode is allowlist"), gomcp.WithStringItems()),
		gomcp.WithNumber("maxPages", gomcp.Description("Maximum pages to crawl (default 10000)")),
		gomcp.WithNumber("maxDepth", gomcp.Description("Maximum link depth (default 50)")),
		gomcp.WithString("renderMode", gomcp.Description("Page rendering strategy"), gomcp.Enum("static", "browser", "hybrid")),
		gomcp.WithBoolean("respectRobots", gomcp.Description("Honor robots.txt directives (default true)")),
		gomcp.WithBoolean("dryRun", gomcp.Description("Fetch seeds only without full crawl (default false)")),
	)

	crawlStatusTool = gomcp.NewTool("crawl_status",
		gomcp.WithDescription("Get the current status and progress of a crawl job."),
		gomcp.WithString("jobId", gomcp.Required(), gomcp.Description("Crawl job ID")),
	)

	cancelCrawlTool = gomcp.NewTool("cancel_crawl",
		gomcp.WithDescription("Cancel a running crawl job."),
		gomcp.WithString("jobId", gomcp.Required(), gomcp.Description("Crawl job ID to cancel")),
	)

	getCrawlSummaryTool = gomcp.NewTool("get_crawl_summary",
		gomcp.WithDescription("Get a high-level summary of crawl results including issue counts and page statistics."),
		gomcp.WithString("jobId", gomcp.Required(), gomcp.Description("Crawl job ID")),
	)

	getCrawlResultsTool = gomcp.NewTool("get_crawl_results",
		gomcp.WithDescription("Query detailed crawl results with filtering and pagination."),
		gomcp.WithString("jobId", gomcp.Required(), gomcp.Description("Crawl job ID")),
		gomcp.WithString("filter", gomcp.Description("Filter expression for results")),
		gomcp.WithNumber("limit", gomcp.Description("Maximum results to return")),
		gomcp.WithString("cursor", gomcp.Description("Pagination cursor")),
	)

	getLinkGraphTool = gomcp.NewTool("get_link_graph",
		gomcp.WithDescription("Get the link graph (edges) for a crawl job."),
		gomcp.WithString("jobId", gomcp.Required(), gomcp.Description("Crawl job ID")),
		gomcp.WithNumber("urlId", gomcp.Description("Filter edges by source URL ID")),
		gomcp.WithString("direction", gomcp.Description("Edge direction"), gomcp.Enum("outbound", "inbound")),
		gomcp.WithNumber("limit", gomcp.Description("Maximum edges to return")),
		gomcp.WithString("cursor", gomcp.Description("Pagination cursor")),
	)

	analyzeURLTool = gomcp.NewTool("analyze_url",
		gomcp.WithDescription("Analyze a single URL for SEO issues without a full crawl."),
		gomcp.WithString("url", gomcp.Required(), gomcp.Description("URL to analyze")),
		gomcp.WithString("renderMode", gomcp.Description("Rendering strategy"), gomcp.Enum("static", "browser")),
	)

	checkRedirectsTool = gomcp.NewTool("check_redirects",
		gomcp.WithDescription("Follow and report the redirect chain for a URL."),
		gomcp.WithString("url", gomcp.Required(), gomcp.Description("URL to check redirects for")),
	)

	checkRobotsTxtTool = gomcp.NewTool("check_robots_txt",
		gomcp.WithDescription("Fetch and parse robots.txt for a given host."),
		gomcp.WithString("url", gomcp.Required(), gomcp.Description("URL whose host's robots.txt to check")),
	)

	parseSitemapTool = gomcp.NewTool("parse_sitemap",
		gomcp.WithDescription("Parse a sitemap XML and return its entries."),
		gomcp.WithString("url", gomcp.Required(), gomcp.Description("Sitemap URL to parse")),
		gomcp.WithNumber("limit", gomcp.Description("Maximum entries to return")),
	)
)

// registerTools adds all tool handlers to the MCP server.
func (s *Server) registerTools() {
	// Crawl lifecycle
	s.mcpServer.AddTool(crawlSiteTool, s.handleCrawlSite)
	s.mcpServer.AddTool(crawlStatusTool, s.handleCrawlStatus)
	s.mcpServer.AddTool(cancelCrawlTool, s.handleCancelCrawl)

	// Query tools
	s.mcpServer.AddTool(getCrawlSummaryTool, s.handleGetCrawlSummary)
	s.mcpServer.AddTool(getCrawlResultsTool, s.handleGetCrawlResults)
	s.mcpServer.AddTool(getLinkGraphTool, s.handleGetLinkGraph)

	// Standalone tools
	s.mcpServer.AddTool(analyzeURLTool, s.handleAnalyzeURL)
	s.mcpServer.AddTool(checkRedirectsTool, s.handleCheckRedirects)
	s.mcpServer.AddTool(checkRobotsTxtTool, s.handleCheckRobotsTxt)
	s.mcpServer.AddTool(parseSitemapTool, s.handleParseSitemap)
}
