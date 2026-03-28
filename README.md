# seo-crawler-mcp

A deterministic SEO spider exposed as an MCP (Model Context Protocol) server. Built in Go with SQLite persistence, hybrid JS rendering, and 10 MCP tools.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2025--06--18-blue)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## Features

- **Deterministic crawling** — each URL fetched exactly once per job, reproducible results
- **Hybrid JS rendering** — static-first with automatic browser escalation on SPA detection
- **44 SEO issue types** — 27 page-local + 17 global cross-page issues
- **Full link graph** — typed edges (hyperlink, canonical, redirect, hreflang, etc.)
- **Discovery** — robots.txt, sitemap (recursive), llms.txt parsing
- **SSRF protection** — 20+ blocked CIDR ranges by default (loopback, link-local, RFC1918, etc.)
- **Cursor-based pagination** — stable, efficient result traversal
- **Single binary** — CGO-free SQLite via modernc.org, zero external dependencies

## Quick Start

```bash
go install github.com/ggonzalezaleman/seo-crawler-mcp@latest

# Or build from source:
git clone https://github.com/ggonzalezaleman/seo-crawler-mcp
cd seo-crawler-mcp
go build -o seo-crawler-mcp .
```

Add to your MCP client (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "seo-crawler": {
      "command": "seo-crawler-mcp",
      "args": ["--db", "/path/to/crawls.db"]
    }
  }
}
```

## MCP Tools

### `crawl_site`

Start a new site crawl. Returns a job ID for tracking progress.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | yes | — | Seed URL to begin crawling |
| `urls` | string[] | no | — | Additional seed URLs |
| `scopeMode` | string | no | `registrable_domain` | `registrable_domain`, `exact_host`, or `allowlist` |
| `allowedHosts` | string[] | no | — | Hosts to allow when scopeMode is `allowlist` |
| `maxPages` | number | no | 10000 | Maximum pages to crawl |
| `maxDepth` | number | no | 50 | Maximum link depth |
| `renderMode` | string | no | `hybrid` | `static`, `browser`, or `hybrid` |
| `respectRobots` | boolean | no | true | Honor robots.txt directives |
| `dryRun` | boolean | no | false | Fetch seeds only without full crawl |

```
crawl_site(url: "https://example.com", maxPages: 500, renderMode: "static")
```

### `crawl_status`

Get the current status and progress of a crawl job.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `jobId` | string | yes | Crawl job ID |

```
crawl_status(jobId: "abc123")
```

### `cancel_crawl`

Cancel a running crawl job.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `jobId` | string | yes | Crawl job ID to cancel |

### `get_crawl_summary`

Get a high-level summary of crawl results including issue counts and page statistics.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `jobId` | string | yes | Crawl job ID |

### `get_crawl_results`

Query detailed crawl results with filtering and pagination.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `jobId` | string | yes | — | Crawl job ID |
| `view` | string | no | `pages` | `pages`, `issues`, `external_links`, or `response_codes` |
| `limit` | number | no | 50 | Max results (max 500) |
| `cursor` | string | no | — | Pagination cursor (base64) |
| `issueType` | string | no | — | Filter by issue type |
| `statusCodeFamily` | string | no | — | Filter: `2xx`, `3xx`, `4xx`, `5xx` |
| `urlPattern` | string | no | — | Filter by URL substring |
| `urlGroup` | string | no | — | Filter by URL group |
| `minDepth` | number | no | — | Filter by minimum depth |
| `maxDepth` | number | no | — | Filter by maximum depth |
| `relationType` | string | no | — | Filter by edge relation type |
| `contentType` | string | no | — | Filter by content type |
| `targetDomain` | string | no | — | Filter by target domain |

```
get_crawl_results(jobId: "abc123", view: "issues", issueType: "missing_title", limit: 100)
```

### `get_link_graph`

Get the link graph (edges) for a crawl job, centered on a specific URL.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `jobId` | string | yes | — | Crawl job ID |
| `urlId` | number | yes | — | URL ID to query edges for |
| `direction` | string | no | `outbound` | `outbound`, `inbound`, or `both` |
| `limit` | number | no | 50 | Max edges (max 500) |
| `cursor` | string | no | — | Pagination cursor |
| `relationType` | string | no | — | Filter by relation type |
| `sourceKind` | string | no | — | Filter by source kind |

```
get_link_graph(jobId: "abc123", urlId: 42, direction: "both")
```

### `analyze_url`

Analyze a single URL for SEO issues without a full crawl.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | yes | — | URL to analyze |
| `renderMode` | string | no | `static` | `static` or `browser` |

```
analyze_url(url: "https://example.com/pricing")
```

### `check_redirects`

Follow and report the redirect chain for a URL.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | yes | — | URL to check |
| `maxHops` | number | no | 10 | Maximum redirect hops |

```
check_redirects(url: "http://example.com")
```

### `check_robots_txt`

Fetch and parse robots.txt for a given host.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | yes | — | URL whose host's robots.txt to check |
| `userAgent` | string | no | — | User-agent to test rules against |
| `testPaths` | string[] | no | — | Paths to test against rules |

```
check_robots_txt(url: "https://example.com", userAgent: "Googlebot", testPaths: ["/admin", "/api"])
```

### `parse_sitemap`

Parse a sitemap XML and return its entries.

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | string | yes | — | Sitemap URL to parse |
| `maxEntries` | number | no | 10000 | Maximum entries to return |

```
parse_sitemap(url: "https://example.com/sitemap.xml")
```

## Resources

| URI | Description |
|-----|-------------|
| `seo-crawler://jobs` | List of all crawl jobs |
| `seo-crawler://jobs/{jobId}` | Job detail with URL/issue counters |
| `seo-crawler://jobs/{jobId}/summary` | Crawl summary snapshot |
| `seo-crawler://jobs/{jobId}/events` | Crawl event log |

## Prompts

### `analyze_technical_seo`

Generates a technical SEO analysis prompt with the crawl summary embedded. Focuses on critical issues, duplicate content, orphan pages, canonical problems, and indexability.

| Argument | Required | Description |
|----------|----------|-------------|
| `jobId` | yes | Crawl job ID |

### `investigate_url`

Generates a deep investigation prompt for a specific URL, including page data, issues, and outbound links.

| Argument | Required | Description |
|----------|----------|-------------|
| `jobId` | yes | Crawl job ID |
| `url` | yes | URL to investigate |

## Issue Types Reference

### Page-Local Issues (27)

| Type | Severity | Description |
|------|----------|-------------|
| `missing_title` | error | Page has no `<title>` tag |
| `title_too_long` | warning | Title exceeds max length (default 60) |
| `title_too_short` | warning | Title below min length (default 30) |
| `missing_description` | error | Page has no meta description |
| `description_too_long` | warning | Description exceeds max length (default 160) |
| `description_too_short` | warning | Description below min length (default 70) |
| `missing_canonical` | warning | No canonical URL declared |
| `missing_h1` | warning | Page has no H1 heading |
| `multiple_h1` | warning | Page has more than one H1 |
| `missing_og_title` | info | No Open Graph title |
| `missing_og_description` | info | No Open Graph description |
| `missing_og_image` | info | No Open Graph image |
| `missing_structured_data` | info | No JSON-LD blocks found |
| `malformed_structured_data` | warning | Invalid JSON-LD syntax |
| `thin_content` | warning | Word count below threshold (default 200) |
| `missing_alt_attribute` | warning | Images missing alt attribute |
| `empty_alt_attribute` | info | Images with empty alt attribute |
| `mixed_content` | warning | HTTP resources on HTTPS page |
| `slow_response` | info | TTFB > 3000ms |
| `very_slow_response` | warning | TTFB > 10000ms |
| `deep_page` | info | Page depth exceeds threshold (default 3) |
| `js_suspect_not_rendered` | info | SPA signals detected, may need browser rendering |
| `status_4xx` | error | HTTP 4xx response |
| `status_5xx` | error | HTTP 5xx response |
| `redirect_chain` | warning | More than 1 redirect hop |
| `redirect_loop` | error | Redirect loop detected |
| `redirect_hops_exceeded` | error | Redirect chain exceeds max hops |

### Global Issues (17)

| Type | Severity | Description |
|------|----------|-------------|
| `duplicate_title` | warning | Multiple pages share the same title |
| `duplicate_description` | warning | Multiple pages share the same meta description |
| `duplicate_content` | warning | Multiple pages share the same content hash |
| `orphan_page` | warning | Page has zero inbound links (not a seed) |
| `deep_page` | info | Page depth exceeds threshold (global detection) |
| `hreflang_not_reciprocal` | warning | Hreflang target doesn't link back |
| `broken_hreflang_target` | error | Hreflang target returns non-200 |
| `canonical_to_non_200` | error | Canonical URL returns non-200 status |
| `canonical_chain` | warning | Canonical points to a page with a different canonical |
| `canonical_to_redirect` | warning | Canonical URL is a redirect (301/302/307/308) |
| `broken_pagination_chain` | warning | rel=next/prev target returns non-200 |
| `pagination_canonical_mismatch` | warning | Paginated page has canonical pointing elsewhere |
| `sitemap_non_200` | warning | Sitemap URL returns non-200 |
| `crawled_not_in_sitemap` | info | Indexable page not found in sitemap |
| `in_sitemap_not_crawled` | info | Sitemap URL not reached during crawl |
| `in_sitemap_robots_blocked` | warning | Sitemap URL blocked by robots.txt |
| `http_to_https_missing` | info | HTTP host has no redirect to HTTPS |

## Configuration

Configuration is loaded with precedence: **environment variables > config file > defaults**.

### Config File (TOML)

```toml
# seo-crawler.toml

# Crawl scope
scope_mode = "registrable_domain"    # registrable_domain | exact_host | allowlist
allowed_hosts = []

# HTTP client
request_timeout = "10s"
max_response_body = 5242880          # 5 MB
max_decompressed_body = 20971520     # 20 MB
user_agent = "seo-crawler-mcp/0.1"
retries = 1
max_redirect_hops = 10

# Concurrency
per_host_concurrency = 2
global_concurrency = 8

# Crawl limits
max_pages = 10000
max_depth = 50

# Rendering
render_mode = "hybrid"               # static | hybrid | browser
render_wait_ms = 2000
max_browser_instances = 2
browser_render_timeout = "30s"
force_render_patterns = []

# Robots
respect_robots = true
robots_unreachable_policy = "allow"  # allow | disallow | cache_then_allow

# URL normalization
ignore_params = ["utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content", "fbclid", "gclid"]
max_query_variants_per_path = 50

# Security
allow_insecure_tls = false
allow_private_networks = false
ssrf_protection = true

# SEO thresholds
title_max_length = 60
title_min_length = 30
description_max_length = 160
description_min_length = 70
thin_content_threshold = 200
deep_page_threshold = 3

# Rate limiting
max_concurrent_crawls = 3
max_concurrent_analyze = 50
max_jobs_per_hour = 20
analyze_job_ttl = "24h"

# Sitemap
max_sitemap_entries = 500000

# Storage
max_queue_memory_mb = 100
db_path = "seo-crawler.db"
max_job_age = "0s"                   # 0 = disabled
```

### Environment Variables

All config options can be overridden with `SEO_CRAWLER_` prefixed environment variables:

| Variable | Type | Description |
|----------|------|-------------|
| `SEO_CRAWLER_DB_PATH` | string | Database path |
| `SEO_CRAWLER_MAX_PAGES` | int | Maximum pages per crawl |
| `SEO_CRAWLER_MAX_DEPTH` | int | Maximum link depth |
| `SEO_CRAWLER_USER_AGENT` | string | HTTP User-Agent |
| `SEO_CRAWLER_SCOPE_MODE` | string | Crawl scope boundary |
| `SEO_CRAWLER_REQUEST_TIMEOUT` | duration | HTTP request timeout |
| `SEO_CRAWLER_MAX_RESPONSE_BODY` | int64 | Max response body bytes |
| `SEO_CRAWLER_MAX_DECOMPRESSED_BODY` | int64 | Max decompressed body bytes |
| `SEO_CRAWLER_RETRIES` | int | Retry count |
| `SEO_CRAWLER_MAX_REDIRECT_HOPS` | int | Max redirect hops |
| `SEO_CRAWLER_PER_HOST_CONCURRENCY` | int | Per-host concurrency |
| `SEO_CRAWLER_GLOBAL_CONCURRENCY` | int | Global concurrency |
| `SEO_CRAWLER_RENDER_MODE` | string | Rendering strategy |
| `SEO_CRAWLER_RENDER_WAIT_MS` | int | Browser render wait |
| `SEO_CRAWLER_MAX_BROWSER_INSTANCES` | int | Browser pool size |
| `SEO_CRAWLER_BROWSER_RENDER_TIMEOUT` | duration | Browser render timeout |
| `SEO_CRAWLER_RESPECT_ROBOTS` | bool | Honor robots.txt |
| `SEO_CRAWLER_ROBOTS_UNREACHABLE_POLICY` | string | Robots.txt fallback policy |
| `SEO_CRAWLER_ALLOW_INSECURE_TLS` | bool | Allow invalid TLS certs |
| `SEO_CRAWLER_ALLOW_PRIVATE_NETWORKS` | bool | Allow private IPs |
| `SEO_CRAWLER_SSRF_PROTECTION` | bool | Enable SSRF guard |
| `SEO_CRAWLER_TITLE_MAX_LENGTH` | int | Title max length threshold |
| `SEO_CRAWLER_TITLE_MIN_LENGTH` | int | Title min length threshold |
| `SEO_CRAWLER_DESCRIPTION_MAX_LENGTH` | int | Description max length |
| `SEO_CRAWLER_DESCRIPTION_MIN_LENGTH` | int | Description min length |
| `SEO_CRAWLER_THIN_CONTENT_THRESHOLD` | int | Thin content word count |
| `SEO_CRAWLER_DEEP_PAGE_THRESHOLD` | int | Deep page depth |
| `SEO_CRAWLER_MAX_CONCURRENT_CRAWLS` | int | Max simultaneous crawls |
| `SEO_CRAWLER_MAX_CONCURRENT_ANALYZE` | int | Max simultaneous analyze jobs |
| `SEO_CRAWLER_MAX_JOBS_PER_HOUR` | int | Rate limit: jobs per hour |
| `SEO_CRAWLER_ANALYZE_JOB_TTL` | duration | Analyze job TTL |
| `SEO_CRAWLER_MAX_SITEMAP_ENTRIES` | int | Max sitemap entries |
| `SEO_CRAWLER_MAX_QUEUE_MEMORY_MB` | int | Frontier queue memory cap |
| `SEO_CRAWLER_MAX_QUERY_VARIANTS_PER_PATH` | int | Max query param variants |

## CLI Usage

```
seo-crawler-mcp [flags] [command]

Flags:
  --version        Print version and exit
  --check-config   Validate and print effective config as JSON
  --config PATH    Path to config file (TOML)
  --db PATH        Path to SQLite database

Commands:
  purge            Remove old crawl data
    --older-than   Duration (e.g., 30d, 24h)
    --job ID       Specific job ID
    --db PATH      Database path (default: seo-crawler.db)
```

Examples:

```bash
# Start MCP server with custom database
seo-crawler-mcp --db /data/crawls.db

# Validate config
seo-crawler-mcp --config seo-crawler.toml --check-config

# Purge jobs older than 30 days
seo-crawler-mcp purge --older-than 30d --db /data/crawls.db

# Purge a specific job
seo-crawler-mcp purge --job abc123
```

## Architecture

```
Seeds → Frontier → [Fetcher Pool] → [Parser Pool] → [Issue Detector] → [Persister] → SQLite
                                                                              ↓
                                                              [Global Issues] → [Materializer]
```

### Package Layout

```
internal/
├── config/       Configuration loading (TOML + env vars)
├── crawl/        Crawl orchestration and site onboarding
├── dto/          Data transfer objects (DB → MCP response conversion)
├── encoding/     Character encoding detection
├── engine/       Crawl engine (coordinator)
├── fetcher/      HTTP client with rate limiting and retry
├── frontier/     URL queue (priority, dedup, memory-bounded)
├── issues/       SEO issue detection (page-local + global)
├── llmstxt/      llms.txt parser
├── materialize/  Post-crawl summary materialization
├── mcp/          MCP server (tools, resources, prompts)
├── parser/       HTML parser (meta tags, headings, JSON-LD, word count)
├── renderer/     Headless browser pool for JS rendering
├── robots/       robots.txt parser
├── sitemap/      Sitemap XML parser (recursive)
├── ssrf/         SSRF guard (CIDR block list)
├── storage/      SQLite persistence layer
├── urlgroup/     URL grouping by pattern
└── urlutil/      URL normalization utilities
```

## Development

```bash
# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Build
go build -o seo-crawler-mcp .
```

## License

[MIT](LICENSE)
