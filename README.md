# seo-crawler-mcp

A deterministic SEO spider exposed as an MCP (Model Context Protocol) server. Built in Go with SQLite persistence, hybrid JS rendering, and 10 MCP tools.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2025--06--18-blue)](https://modelcontextprotocol.io)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## Features

- **103 SEO issue types** — page-local + global cross-page + text quality
- **Hybrid JS rendering** — static-first with automatic Playwright browser escalation
- **Lazy load detection** — full-page scroll to capture intersection-observer content
- **PageSpeed Insights** — mobile + desktop audits with all Lighthouse categories
- **Axe accessibility** — WCAG 2.x violations and passes per page
- **LanguageTool** — deterministic spelling/grammar/style checks
- **Full link graph** — typed edges (hyperlink, canonical, redirect, hreflang, etc.)
- **Discovery** — robots.txt, sitemap (recursive), llms.txt parsing
- **SSRF protection** — 20+ blocked CIDR ranges
- **Single binary** — CGO-free SQLite, zero external dependencies

## Quick Start

```bash
# Install
go install github.com/ggonzalezaleman/seo-crawler-mcp@latest

# Or build from source
git clone https://github.com/ggonzalezaleman/seo-crawler-mcp
cd seo-crawler-mcp
go build -o seo-crawler-mcp .
```

### Configure

```bash
# Copy example configs
cp .env.example .env
cp config.example.toml config.toml

# Edit .env with your settings
```

### Optional services

```bash
# LanguageTool (spelling/grammar checks)
docker compose up -d languagetool

# Playwright (JS rendering + accessibility)
pip install playwright && playwright install chromium
```

### Run

```bash
# Start MCP server
seo-crawler-mcp --db crawls.db

# With config file
seo-crawler-mcp --config config.toml --db crawls.db
```

### Add to your MCP client

```json
{
  "mcpServers": {
    "seo-crawler": {
      "command": "seo-crawler-mcp",
      "args": ["--db", "/path/to/crawls.db"],
      "env": {
        "SEO_CRAWLER_PSI_API_KEY": "your_key",
        "SEO_CRAWLER_LANGUAGETOOL_URL": "http://localhost:8010",
        "SEO_CRAWLER_PSI_DESKTOP": "true"
      }
    }
  }
}
```

## Configuration

**Precedence:** environment variables > config file (TOML) > defaults.

### Key environment variables

| Variable | Description |
|----------|-------------|
| `SEO_CRAWLER_PSI_API_KEY` | Google PageSpeed Insights API key ([get one free](https://developers.google.com/speed/docs/insights/v5/get-started)) |
| `SEO_CRAWLER_PSI_DESKTOP` | Enable desktop PSI audits alongside mobile (`true`/`false`) |
| `SEO_CRAWLER_LANGUAGETOOL_URL` | LanguageTool server URL (e.g. `http://localhost:8010`) |
| `SEO_CRAWLER_DB_PATH` | SQLite database path |
| `SEO_CRAWLER_MAX_PAGES` | Max pages per crawl (default: 10000) |
| `SEO_CRAWLER_RENDER_MODE` | `static`, `hybrid`, or `browser` (default: `hybrid`) |
| `SEO_CRAWLER_SCOPE_MODE` | `registrable_domain`, `exact_host`, or `allowlist` |

See [`.env.example`](.env.example) for all options and [`config.example.toml`](config.example.toml) for the full TOML reference.

## MCP Tools

| Tool | Description |
|------|-------------|
| `crawl_site` | Start a new site crawl |
| `crawl_status` | Check crawl progress |
| `cancel_crawl` | Cancel a running crawl |
| `get_crawl_summary` | High-level crawl statistics |
| `get_crawl_results` | Query pages, issues, links, response codes with filters |
| `get_link_graph` | Internal/external link graph per URL |
| `analyze_url` | Single-URL SEO analysis without full crawl |
| `check_redirects` | Follow and report redirect chains |
| `check_robots_txt` | Parse robots.txt and test paths |
| `parse_sitemap` | Parse sitemap XML entries |

## Crawl Pipeline

```
Seeds → Frontier → Fetcher → Parser → Issue Detector → Persister → SQLite
                                                              ↓
                                              Browser Enrich (lazy load)
                                              ↓
                                              Sitemap Gap Escalation
                                              ↓
                                              HEAD-check Assets
                                              ↓
                                              Text Quality (LanguageTool)
                                              ↓
                                              PSI + Axe Audits
                                              ↓
                                              Global Issues → Materializer
                                              ↓
                                              Depth Recomputation (shortest path)
```

### Post-crawl enrichment

| Phase | Requires | What it does |
|-------|----------|-------------|
| **Browser enrich** | Playwright | Re-renders JS-suspect/thin pages with full scroll to capture lazy-loaded content |
| **Sitemap gap escalation** | Playwright | Clicks menus to discover JS-only navigation for sitemap gap URLs |
| **Asset HEAD checks** | — | Verifies status codes of images, scripts, stylesheets, fonts |
| **Text quality** | LanguageTool | Spelling, grammar, style checks on page text |
| **PSI audits** | API key | PageSpeed Insights scores + Core Web Vitals per page |
| **Axe audits** | Playwright | WCAG accessibility violations and passes per page |
| **Global issues** | — | Cross-page: duplicates, orphans, hreflang, canonical chains |
| **Depth recomputation** | — | BFS shortest-path from seeds over final link graph |

## Issue Types (103+)

### Page-local
`missing_title`, `title_too_long`, `title_too_short`, `missing_description`, `description_too_long`, `description_too_short`, `missing_canonical`, `missing_h1`, `multiple_h1`, `missing_og_title`, `missing_og_description`, `missing_og_image`, `missing_structured_data`, `malformed_structured_data`, `invalid_structured_data`, `incomplete_structured_data`, `thin_content`, `missing_alt_attribute`, `empty_alt_attribute`, `mixed_content`, `slow_response`, `very_slow_response`, `deep_page`, `js_suspect_not_rendered`, `status_4xx`, `status_5xx`, `redirect_chain`, `redirect_loop`, and more.

### Global
`duplicate_title`, `duplicate_description`, `duplicate_content`, `orphan_page`, `hreflang_not_reciprocal`, `broken_hreflang_target`, `canonical_to_non_200`, `canonical_chain`, `canonical_to_redirect`, `crawled_not_in_sitemap`, `in_sitemap_not_crawled`, `js_only_navigation`, and more.

### Text quality
`text_quality_spelling`, `text_quality_grammar`, `text_quality_style`, `text_quality_punctuation`

### Security
`missing_content_security_policy`, `missing_referrer_policy`, `missing_permissions_policy`, `missing_x_content_type_options`, and more.

## Development

```bash
# Tests
go test ./... -count=1

# With race detector
go test -race ./...

# Lint
go vet ./...

# Security audit
semgrep --config p/golang --config p/security-audit

# Build
go build -o seo-crawler-mcp .
```

## License

[MIT](LICENSE)
