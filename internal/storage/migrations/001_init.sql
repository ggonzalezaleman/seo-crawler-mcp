-- 001_init.sql — Full schema for seo-crawler-mcp storage layer.

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- ============================================================
-- crawl_jobs: top-level crawl session
-- ============================================================
CREATE TABLE IF NOT EXISTS crawl_jobs (
    id              TEXT    NOT NULL PRIMARY KEY,
    type            TEXT    NOT NULL DEFAULT 'crawl',
    status          TEXT    NOT NULL DEFAULT 'queued',
    config_json     TEXT    NOT NULL DEFAULT '{}',
    seed_urls       TEXT    NOT NULL DEFAULT '[]',
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    started_at      TEXT,
    finished_at     TEXT,
    error           TEXT,
    pages_crawled   INTEGER NOT NULL DEFAULT 0,
    urls_discovered INTEGER NOT NULL DEFAULT 0,
    issues_found    INTEGER NOT NULL DEFAULT 0,
    ttl_expires_at  TEXT
);

CREATE INDEX IF NOT EXISTS idx_crawl_jobs_status ON crawl_jobs(status);

-- ============================================================
-- urls: every URL discovered during a crawl
-- ============================================================
CREATE TABLE IF NOT EXISTS urls (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id          TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    normalized_url  TEXT    NOT NULL,
    host            TEXT    NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'pending',
    is_internal     INTEGER NOT NULL DEFAULT 1,
    discovered_via  TEXT    NOT NULL DEFAULT 'seed',
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE(job_id, normalized_url)
);

CREATE INDEX IF NOT EXISTS idx_urls_job_status ON urls(job_id, status);
CREATE INDEX IF NOT EXISTS idx_urls_job_host   ON urls(job_id, host);

-- ============================================================
-- fetches: HTTP fetch attempts
-- ============================================================
CREATE TABLE IF NOT EXISTS fetches (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id                TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    fetch_seq             INTEGER NOT NULL,
    requested_url_id      INTEGER NOT NULL REFERENCES urls(id),
    final_url_id          INTEGER REFERENCES urls(id),
    status_code           INTEGER,
    redirect_hop_count    INTEGER NOT NULL DEFAULT 0,
    ttfb_ms               INTEGER,
    response_body_size    INTEGER,
    content_type          TEXT,
    content_encoding      TEXT,
    response_headers_json TEXT,
    http_method           TEXT    NOT NULL DEFAULT 'GET',
    fetch_kind            TEXT    NOT NULL DEFAULT 'page',
    render_mode           TEXT    NOT NULL DEFAULT 'static',
    render_params_json    TEXT,
    fetched_at            TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    error                 TEXT,
    UNIQUE(job_id, fetch_seq)
);

CREATE INDEX IF NOT EXISTS idx_fetches_job_seq      ON fetches(job_id, fetch_seq);
CREATE INDEX IF NOT EXISTS idx_fetches_requested_url ON fetches(requested_url_id);

-- ============================================================
-- pages: parsed page-level SEO data
-- ============================================================
CREATE TABLE IF NOT EXISTS pages (
    id                        INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id                    TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    url_id                    INTEGER NOT NULL REFERENCES urls(id),
    fetch_id                  INTEGER NOT NULL REFERENCES fetches(id),
    depth                     INTEGER NOT NULL DEFAULT 0,
    title                     TEXT,
    title_length              INTEGER,
    meta_description          TEXT,
    meta_description_length   INTEGER,
    meta_robots               TEXT,
    x_robots_tag              TEXT,
    indexability_state         TEXT    NOT NULL DEFAULT 'unknown',
    canonical_url             TEXT,
    canonical_is_self         INTEGER,
    canonical_status_code     INTEGER,
    rel_next_url              TEXT,
    rel_prev_url              TEXT,
    hreflang_json             TEXT,
    h1_json                   TEXT,
    h2_json                   TEXT,
    h3_json                   TEXT,
    h4_json                   TEXT,
    h5_json                   TEXT,
    h6_json                   TEXT,
    og_title                  TEXT,
    og_description            TEXT,
    og_image                  TEXT,
    og_url                    TEXT,
    og_type                   TEXT,
    twitter_card              TEXT,
    twitter_title             TEXT,
    twitter_description       TEXT,
    twitter_image             TEXT,
    jsonld_raw                TEXT,
    jsonld_types_json         TEXT,
    images_json               TEXT,
    word_count                INTEGER,
    main_content_word_count   INTEGER,
    content_hash              TEXT,
    js_suspect                INTEGER NOT NULL DEFAULT 0,
    url_group                 TEXT,
    outbound_edge_count       INTEGER NOT NULL DEFAULT 0,
    inbound_edge_count        INTEGER NOT NULL DEFAULT 0,
    UNIQUE(job_id, url_id)
);

CREATE INDEX IF NOT EXISTS idx_pages_job_url   ON pages(job_id, url_id);
CREATE INDEX IF NOT EXISTS idx_pages_job_group ON pages(job_id, url_group);

-- ============================================================
-- edges: link relationships between pages
-- ============================================================
CREATE TABLE IF NOT EXISTS edges (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id                   TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    source_url_id            INTEGER NOT NULL REFERENCES urls(id),
    normalized_target_url_id INTEGER REFERENCES urls(id),
    source_kind              TEXT    NOT NULL DEFAULT 'html',
    relation_type            TEXT    NOT NULL DEFAULT 'hyperlink',
    rel_flags_json           TEXT,
    discovery_mode           TEXT    NOT NULL DEFAULT 'crawl',
    anchor_text              TEXT,
    is_internal              INTEGER NOT NULL DEFAULT 1,
    declared_target_url      TEXT    NOT NULL,
    final_target_url_id      INTEGER REFERENCES urls(id),
    target_status_code       INTEGER
);

CREATE INDEX IF NOT EXISTS idx_edges_job_source ON edges(job_id, source_url_id);
CREATE INDEX IF NOT EXISTS idx_edges_job_target ON edges(job_id, normalized_target_url_id);
CREATE INDEX IF NOT EXISTS idx_edges_job ON edges(job_id);

-- ============================================================
-- redirect_hops: individual hops in a redirect chain
-- ============================================================
CREATE TABLE IF NOT EXISTS redirect_hops (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id     TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    fetch_id   INTEGER NOT NULL REFERENCES fetches(id) ON DELETE CASCADE,
    hop_index  INTEGER NOT NULL,
    status_code INTEGER NOT NULL,
    from_url   TEXT    NOT NULL,
    to_url     TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_redirect_hops_fetch ON redirect_hops(fetch_id);

-- ============================================================
-- sitemap_entries: URLs found in sitemaps
-- ============================================================
CREATE TABLE IF NOT EXISTS sitemap_entries (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id                 TEXT NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    url                    TEXT NOT NULL,
    source_sitemap_url     TEXT NOT NULL,
    source_host            TEXT NOT NULL,
    lastmod                TEXT,
    changefreq             TEXT,
    priority               REAL,
    reconciliation_status  TEXT NOT NULL DEFAULT 'pending'
);

CREATE INDEX IF NOT EXISTS idx_sitemap_entries_job_url ON sitemap_entries(job_id, url);

-- ============================================================
-- robots_directives: parsed robots.txt rules
-- ============================================================
CREATE TABLE IF NOT EXISTS robots_directives (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id       TEXT NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    host         TEXT NOT NULL,
    user_agent   TEXT NOT NULL,
    rule_type    TEXT NOT NULL,
    path_pattern TEXT NOT NULL,
    source_url   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_robots_directives_job_host ON robots_directives(job_id, host);

-- ============================================================
-- llms_findings: llms.txt analysis per host
-- ============================================================
CREATE TABLE IF NOT EXISTS llms_findings (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id              TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    host                TEXT    NOT NULL,
    present             INTEGER NOT NULL DEFAULT 0,
    raw_content         TEXT,
    sections_json       TEXT,
    referenced_urls_json TEXT,
    UNIQUE(job_id, host)
);

-- ============================================================
-- assets: external resources (CSS, JS, images, fonts)
-- ============================================================
CREATE TABLE IF NOT EXISTS assets (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id         TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    url_id         INTEGER NOT NULL REFERENCES urls(id),
    content_type   TEXT,
    status_code    INTEGER,
    content_length INTEGER
);

CREATE INDEX IF NOT EXISTS idx_assets_job_url ON assets(job_id, url_id);

-- ============================================================
-- asset_references: which pages reference which assets
-- ============================================================
CREATE TABLE IF NOT EXISTS asset_references (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id              TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    asset_url_id        INTEGER NOT NULL REFERENCES urls(id),
    source_page_url_id  INTEGER NOT NULL REFERENCES urls(id),
    reference_type      TEXT    NOT NULL DEFAULT 'link'
);

CREATE INDEX IF NOT EXISTS idx_asset_refs_job_asset ON asset_references(job_id, asset_url_id);

-- ============================================================
-- issues: detected SEO issues
-- ============================================================
CREATE TABLE IF NOT EXISTS issues (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id       TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    url_id       INTEGER REFERENCES urls(id),
    issue_type   TEXT    NOT NULL,
    severity     TEXT    NOT NULL DEFAULT 'warning',
    scope        TEXT    NOT NULL DEFAULT 'page',
    details_json TEXT
);

CREATE INDEX IF NOT EXISTS idx_issues_job_type     ON issues(job_id, issue_type);
CREATE INDEX IF NOT EXISTS idx_issues_job_severity ON issues(job_id, severity);

-- ============================================================
-- crawl_events: timeline of crawl activity
-- ============================================================
CREATE TABLE IF NOT EXISTS crawl_events (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id       TEXT NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    timestamp    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    event_type   TEXT NOT NULL,
    details_json TEXT,
    url          TEXT
);

CREATE INDEX IF NOT EXISTS idx_crawl_events_job_type ON crawl_events(job_id, event_type);

-- ============================================================
-- url_pattern_groups: URL grouping patterns
-- ============================================================
CREATE TABLE IF NOT EXISTS url_pattern_groups (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id  TEXT NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    pattern TEXT NOT NULL,
    name    TEXT NOT NULL,
    source  TEXT NOT NULL DEFAULT 'auto'
);

-- ============================================================
-- canonical_clusters: groups of URLs sharing a canonical
-- ============================================================
CREATE TABLE IF NOT EXISTS canonical_clusters (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id             TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    cluster_url        TEXT    NOT NULL,
    member_count       INTEGER NOT NULL DEFAULT 0,
    target_status_code INTEGER,
    is_self_referencing INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_canonical_clusters_job ON canonical_clusters(job_id);

-- ============================================================
-- canonical_cluster_members
-- ============================================================
CREATE TABLE IF NOT EXISTS canonical_cluster_members (
    cluster_id INTEGER NOT NULL REFERENCES canonical_clusters(id) ON DELETE CASCADE,
    url_id     INTEGER NOT NULL REFERENCES urls(id),
    job_id     TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    PRIMARY KEY (cluster_id, url_id)
);

-- ============================================================
-- duplicate_clusters: content/near-duplicate groups
-- ============================================================
CREATE TABLE IF NOT EXISTS duplicate_clusters (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id       TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    cluster_type TEXT    NOT NULL,
    hash_value   TEXT    NOT NULL,
    first_url_id INTEGER NOT NULL REFERENCES urls(id),
    member_count INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_duplicate_clusters_job ON duplicate_clusters(job_id);

-- ============================================================
-- duplicate_cluster_members
-- ============================================================
CREATE TABLE IF NOT EXISTS duplicate_cluster_members (
    cluster_id INTEGER NOT NULL REFERENCES duplicate_clusters(id) ON DELETE CASCADE,
    url_id     INTEGER NOT NULL REFERENCES urls(id),
    job_id     TEXT    NOT NULL REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    fetch_seq  INTEGER,
    PRIMARY KEY (cluster_id, url_id)
);
