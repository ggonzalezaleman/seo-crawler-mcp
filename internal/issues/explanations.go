package issues

// IssueExplanation provides human-readable context for each issue type.
type IssueExplanation struct {
	Title       string // Human-readable title (e.g., "Invalid Structured Data")
	Description string // What it means (1-2 sentences)
	Impact      string // Why it matters for SEO (1 sentence)
	Fix         string // How to fix it (1 sentence)
}

// Explanations maps every known issue type string to its human-readable explanation.
var Explanations = map[string]IssueExplanation{
	// ── Page-local issues (from DetectPageLocalIssues) ──────────────────

	"missing_title": {
		Title:       "Missing Page Title",
		Description: "This page has no <title> tag in the HTML head.",
		Impact:      "Search engines use the title tag as the primary headline in search results. Without it, Google will auto-generate one, often poorly.",
		Fix:         "Add a unique, descriptive <title> tag between 30-60 characters.",
	},
	"title_too_short": {
		Title:       "Title Too Short",
		Description: "The page title is under 30 characters.",
		Impact:      "Short titles waste valuable SERP real estate and may not adequately describe the page content to users and search engines.",
		Fix:         "Expand the title to 30-60 characters with relevant keywords and a compelling description.",
	},
	"title_too_long": {
		Title:       "Title Too Long",
		Description: "The page title exceeds 60 characters.",
		Impact:      "Google typically truncates titles at ~60 characters in search results, meaning your full message won't be visible.",
		Fix:         "Trim the title to under 60 characters, front-loading the most important keywords.",
	},
	"missing_description": {
		Title:       "Missing Meta Description",
		Description: "This page has no meta description tag.",
		Impact:      "Without a meta description, search engines will auto-generate a snippet from page content, which may not be compelling or relevant.",
		Fix:         "Add a meta description between 70-160 characters that summarizes the page and includes a call to action.",
	},
	"description_too_short": {
		Title:       "Meta Description Too Short",
		Description: "The meta description is under 70 characters.",
		Impact:      "Short descriptions don't fully utilize the space available in search results, missing an opportunity to attract clicks.",
		Fix:         "Expand to 70-160 characters with a compelling summary and relevant keywords.",
	},
	"description_too_long": {
		Title:       "Meta Description Too Long",
		Description: "The meta description exceeds 160 characters.",
		Impact:      "Google truncates descriptions beyond ~160 characters, so the end of your message will be cut off in search results.",
		Fix:         "Trim to under 160 characters, keeping the most important information first.",
	},
	"missing_canonical": {
		Title:       "Missing Canonical Tag",
		Description: "This page does not declare a canonical URL.",
		Impact:      "Without a canonical tag, search engines may split ranking signals across duplicate or similar URLs, diluting page authority.",
		Fix:         "Add <link rel=\"canonical\" href=\"...\"> pointing to the preferred URL for this content.",
	},
	"missing_h1": {
		Title:       "Missing H1 Heading",
		Description: "This page has no <h1> tag.",
		Impact:      "The H1 is the primary heading that tells search engines and users what the page is about. Missing it weakens content signals.",
		Fix:         "Add a single, descriptive H1 heading that includes the page's primary keyword.",
	},
	"multiple_h1": {
		Title:       "Multiple H1 Headings",
		Description: "This page has more than one <h1> tag.",
		Impact:      "Multiple H1s dilute the primary topic signal and confuse the content hierarchy for search engines.",
		Fix:         "Keep a single H1 per page. Demote additional headings to H2 or H3.",
	},
	"thin_content": {
		Title:       "Thin Content",
		Description: "The page's main content has fewer words than the configured threshold (default: 200).",
		Impact:      "Pages with very little content are less likely to rank well, as search engines may consider them low-value.",
		Fix:         "Add substantive, relevant content. If the page is intentionally brief (e.g., a contact form), consider noindexing it.",
	},
	"missing_alt_attribute": {
		Title:       "Images Missing Alt Text",
		Description: "One or more images on this page have no alt attribute.",
		Impact:      "Alt text is critical for accessibility and helps search engines understand image content. Missing it hurts image SEO and accessibility compliance.",
		Fix:         "Add descriptive alt text to every image that conveys meaningful content.",
	},
	"empty_alt_attribute": {
		Title:       "Images With Empty Alt Text",
		Description: "One or more images have alt=\"\" (empty alt attribute).",
		Impact:      "Empty alt is valid for decorative images, but if these images convey content, they're invisible to screen readers and search engines.",
		Fix:         "If the image is decorative, empty alt is correct. If it conveys meaning, add descriptive alt text.",
	},
	"missing_og_title": {
		Title:       "Missing Open Graph Title",
		Description: "No og:title meta tag found.",
		Impact:      "Social media platforms use og:title for link previews. Without it, shares may show incorrect or generic titles.",
		Fix:         "Add <meta property=\"og:title\" content=\"...\"> matching or complementing the page title.",
	},
	"missing_og_description": {
		Title:       "Missing Open Graph Description",
		Description: "No og:description meta tag found.",
		Impact:      "Social shares will use auto-generated descriptions, which are often irrelevant or poorly formatted.",
		Fix:         "Add <meta property=\"og:description\" content=\"...\"> with a compelling social-specific description.",
	},
	"missing_og_image": {
		Title:       "Missing Open Graph Image",
		Description: "No og:image meta tag found.",
		Impact:      "Links shared on social media without an og:image appear as plain text links, dramatically reducing engagement.",
		Fix:         "Add <meta property=\"og:image\" content=\"...\"> with an image at least 1200x630px.",
	},
	"missing_structured_data": {
		Title:       "No Structured Data Found",
		Description: "This page has no JSON-LD structured data.",
		Impact:      "Structured data enables rich results in Google (stars, FAQs, breadcrumbs, etc.), giving your listing more visibility.",
		Fix:         "Add relevant JSON-LD markup (Organization, Article, Product, FAQ, etc.) based on the page type.",
	},
	"malformed_structured_data": {
		Title:       "Malformed Structured Data",
		Description: "A JSON-LD block on this page failed to parse as valid JSON.",
		Impact:      "Broken JSON-LD is completely ignored by search engines, as if it doesn't exist.",
		Fix:         "Validate the JSON-LD block with Google's Rich Results Test and fix syntax errors.",
	},
	"invalid_structured_data": {
		Title:       "Invalid Structured Data",
		Description: "A JSON-LD block is missing required properties for its declared @type.",
		Impact:      "Incomplete structured data may not qualify for rich results. Google requires specific properties per schema type.",
		Fix:         "Add the missing required properties. Check Google's structured data documentation for the specific @type.",
	},
	"incomplete_structured_data": {
		Title:       "Incomplete Structured Data",
		Description: "A JSON-LD block is missing recommended (but not required) properties for its @type.",
		Impact:      "While not blocking, adding recommended properties increases the chance of enhanced rich result features.",
		Fix:         "Add the missing recommended properties to maximize rich result eligibility.",
	},
	"mixed_content": {
		Title:       "Mixed Content (HTTP on HTTPS)",
		Description: "This HTTPS page references resources over HTTP.",
		Impact:      "Browsers may block mixed content, breaking functionality. It also signals a security concern to both users and search engines.",
		Fix:         "Update all resource URLs to use HTTPS.",
	},
	"robots_meta_header_mismatch": {
		Title:       "Robots Meta/Header Conflict",
		Description: "The meta robots tag and X-Robots-Tag HTTP header give conflicting directives.",
		Impact:      "Conflicting signals may cause unpredictable indexing behavior; search engines typically apply the most restrictive directive.",
		Fix:         "Align the meta robots tag and X-Robots-Tag header to use the same directives.",
	},
	"status_4xx": {
		Title:       "Client Error (4xx)",
		Description: "This URL returned a 4xx HTTP status code.",
		Impact:      "4xx pages waste crawl budget and create dead ends for users. If linked internally, they pass negative signals.",
		Fix:         "Fix or redirect the URL, and update or remove internal links pointing to it.",
	},
	"status_5xx": {
		Title:       "Server Error (5xx)",
		Description: "This URL returned a 5xx HTTP status code.",
		Impact:      "Server errors prevent indexing entirely. Persistent 5xx responses will cause Google to drop the page from search results.",
		Fix:         "Investigate and resolve the server-side error. Check application logs.",
	},
	"redirect_chain": {
		Title:       "Redirect Chain",
		Description: "This URL goes through multiple redirects before reaching the final destination.",
		Impact:      "Each redirect hop adds latency and may lose a small amount of link equity. Long chains also waste crawl budget.",
		Fix:         "Update links to point directly to the final destination URL, eliminating intermediate redirects.",
	},
	"redirect_loop": {
		Title:       "Redirect Loop",
		Description: "This URL is part of a circular redirect chain that never resolves.",
		Impact:      "The page is completely inaccessible; users get an error and search engines can't crawl it.",
		Fix:         "Break the redirect loop by updating one of the redirects to point to a valid, non-redirecting URL.",
	},
	"redirect_hops_exceeded": {
		Title:       "Too Many Redirect Hops",
		Description: "The redirect chain exceeded the maximum number of allowed hops (default: 10).",
		Impact:      "Browsers and search engine crawlers will give up following the chain, making the page unreachable.",
		Fix:         "Simplify the redirect chain to 1-2 hops maximum.",
	},
	"very_slow_response": {
		Title:       "Very Slow Response",
		Description: "The server's time to first byte (TTFB) exceeded 10 seconds for this page.",
		Impact:      "Extremely slow responses degrade user experience and are a strong negative signal for search rankings.",
		Fix:         "Investigate server performance, caching, database queries, and CDN configuration.",
	},
	"slow_response": {
		Title:       "Slow Response",
		Description: "The server's time to first byte (TTFB) exceeded 3 seconds for this page.",
		Impact:      "Slow server response degrades user experience and can negatively impact search rankings (Core Web Vitals).",
		Fix:         "Investigate server performance: check hosting, caching, database queries, and CDN configuration.",
	},
	"deep_page": {
		Title:       "Deep Page",
		Description: "This page is buried deep in the site hierarchy (many clicks from the homepage).",
		Impact:      "Deep pages get crawled less frequently and may be perceived as less important by search engines.",
		Fix:         "Reduce crawl depth by adding internal links from higher-level pages or improving site navigation.",
	},
	"title_outside_head": {
		Title:       "Title Outside <head>",
		Description: "The <title> element appears outside the <head> section, likely inside <body>.",
		Impact:      "Search engines may ignore the title if it's not in <head>. Google often still recognizes it, but this should not be relied upon.",
		Fix:         "Move the <title> element into the <head> section of the HTML document.",
	},
	"meta_robots_outside_head": {
		Title:       "Meta Robots Outside <head>",
		Description: "A <meta name=\"robots\"> tag appears outside the <head> section.",
		Impact:      "Search engines may ignore robots directives outside <head>. Critical directives like noindex could be missed.",
		Fix:         "Move the <meta name=\"robots\"> tag into the <head> section of the HTML document.",
	},
	"js_suspect_not_rendered": {
		Title:       "Suspected JavaScript-Rendered Content",
		Description: "This page appears to rely heavily on JavaScript for rendering its main content.",
		Impact:      "Search engines may not fully render JS content, meaning important text or links could be invisible to crawlers.",
		Fix:         "Implement server-side rendering (SSR) or pre-rendering for critical content.",
	},

	// ── Global issues (from DetectGlobalIssues) ─────────────────────────

	"duplicate_title": {
		Title:       "Duplicate Page Title",
		Description: "Multiple pages share the same <title> tag text.",
		Impact:      "Duplicate titles confuse search engines about which page to rank, and make search results look repetitive.",
		Fix:         "Give each page a unique, descriptive title that reflects its specific content.",
	},
	"duplicate_description": {
		Title:       "Duplicate Meta Description",
		Description: "Multiple pages share the same meta description.",
		Impact:      "Duplicate descriptions reduce click-through differentiation and signal low-quality content to search engines.",
		Fix:         "Write unique meta descriptions for each page that accurately summarize its content.",
	},
	"duplicate_content": {
		Title:       "Duplicate Content",
		Description: "Multiple pages have substantially identical body content.",
		Impact:      "Search engines may filter duplicate pages from results, wasting crawl budget and diluting ranking signals.",
		Fix:         "Consolidate duplicate pages with canonical tags, 301 redirects, or by differentiating the content.",
	},
	"orphan_page": {
		Title:       "Orphan Page",
		Description: "This page has no internal links pointing to it from other crawled pages.",
		Impact:      "Orphan pages are hard for search engines to discover and are treated as low-priority for crawling and indexing.",
		Fix:         "Add internal links from relevant pages, navigation menus, or sitemaps.",
	},
	"hreflang_not_reciprocal": {
		Title:       "Hreflang Not Reciprocal",
		Description: "This page declares an hreflang alternate, but the target page doesn't link back.",
		Impact:      "Non-reciprocal hreflang tags are ignored by Google, meaning your language/region targeting won't work.",
		Fix:         "Ensure both pages in the hreflang pair reference each other with matching hreflang tags.",
	},
	"broken_hreflang_target": {
		Title:       "Broken Hreflang Target",
		Description: "An hreflang alternate URL returns a non-200 status code.",
		Impact:      "Hreflang pointing to broken pages is ignored, and the language/region targeting fails entirely.",
		Fix:         "Fix the target URL to return 200, or update the hreflang to point to a valid page.",
	},
	"canonical_to_non_200": {
		Title:       "Canonical Points to Non-200",
		Description: "The canonical URL for this page returns a non-200 HTTP status.",
		Impact:      "A canonical pointing to a broken page sends confusing signals; search engines may ignore it and pick their own canonical.",
		Fix:         "Update the canonical tag to point to a valid, 200-status URL.",
	},
	"canonical_chain": {
		Title:       "Canonical Chain",
		Description: "The canonical URL itself has a different canonical, creating a chain.",
		Impact:      "Search engines may not follow canonical chains reliably, so the intended canonical may not be recognized.",
		Fix:         "Point the canonical directly to the final preferred URL, not through an intermediate page.",
	},
	"canonical_to_redirect": {
		Title:       "Canonical Points to Redirect",
		Description: "The canonical URL returns a 3xx redirect instead of a 200.",
		Impact:      "Canonicals should point to the final URL. Pointing to a redirect adds unnecessary indirection and may be ignored.",
		Fix:         "Update the canonical tag to point to the redirect's final destination URL.",
	},
	"broken_pagination_chain": {
		Title:       "Broken Pagination Chain",
		Description: "A rel=next/prev pagination link points to a non-200 page.",
		Impact:      "Broken pagination prevents search engines from discovering and indexing all pages in a paginated series.",
		Fix:         "Fix the target URL or update the pagination links to point to valid pages.",
	},
	"pagination_canonical_mismatch": {
		Title:       "Pagination Canonical Mismatch",
		Description: "A paginated page's canonical URL doesn't match its own URL.",
		Impact:      "If a paginated page canonicalizes to page 1, search engines may ignore pages 2+ entirely.",
		Fix:         "Each paginated page should have a self-referencing canonical, or use a view-all canonical if appropriate.",
	},
	"sitemap_non_200": {
		Title:       "Sitemap URL Returns Non-200",
		Description: "A URL listed in the sitemap returns a non-200 HTTP status.",
		Impact:      "Non-200 URLs in the sitemap waste crawl budget and signal poor site maintenance to search engines.",
		Fix:         "Remove non-200 URLs from the sitemap, or fix the underlying pages.",
	},
	"crawled_not_in_sitemap": {
		Title:       "Crawled Page Not in Sitemap",
		Description: "This indexable page was found by crawling but is not listed in the sitemap.",
		Impact:      "Pages missing from the sitemap may be crawled less frequently and could be deprioritized for indexing.",
		Fix:         "Add all important, indexable pages to the sitemap.",
	},
	"in_sitemap_not_crawled": {
		Title:       "Sitemap URL Not Crawled",
		Description: "A URL in the sitemap was not discovered or reached during the crawl.",
		Impact:      "This could indicate the page is not linked from anywhere on the site (orphan) or the crawl was too shallow.",
		Fix:         "Verify the URL is accessible and internally linked. Increase crawl depth if needed.",
	},
	"in_sitemap_robots_blocked": {
		Title:       "Sitemap URL Blocked by Robots",
		Description: "A URL listed in the sitemap is disallowed by robots.txt.",
		Impact:      "Contradictory signals: the sitemap says 'index this' but robots.txt says 'don't crawl.' Search engines will not index it.",
		Fix:         "Either remove the URL from the sitemap or remove the robots.txt disallow rule.",
	},
	"js_only_navigation": {
		Title:       "JS-Only Navigation Link",
		Description: "This internal link is only visible after JavaScript rendering, not in the static HTML source.",
		Impact:      "Search engines that don't execute JavaScript (or execute it poorly) won't discover this link. This reduces crawl efficiency and may prevent linked pages from being indexed.",
		Fix:         "Ensure navigation links use standard <a href> tags in the server-rendered HTML. For Next.js, verify the links are rendered server-side, not client-only.",
	},
	"http_to_https_missing": {
		Title:       "Missing HTTP to HTTPS Redirect",
		Description: "The site's HTTP version does not redirect to HTTPS.",
		Impact:      "Without an HTTP-to-HTTPS redirect, search engines may index the insecure version, splitting ranking signals.",
		Fix:         "Configure a 301 redirect from HTTP to HTTPS at the server level.",
	},

	// ── Engine-level issues (from crawler engine) ───────────────────────

	"crawl_trap_suspected": {
		Title:       "Suspected Crawl Trap",
		Description: "This URL pattern generated an unusually high number of query string variants.",
		Impact:      "Crawl traps waste crawl budget on infinite URL variations that don't contain unique content.",
		Fix:         "Use rel=canonical, robots.txt, or URL parameter handling in Google Search Console to manage these URLs.",
	},
	"rate_limited": {
		Title:       "Rate Limited (429)",
		Description: "The server responded with HTTP 429 Too Many Requests.",
		Impact:      "Rate limiting indicates the server can't handle the crawl rate. Continued aggressive crawling may lead to IP blocking.",
		Fix:         "This is informational. The crawler automatically backs off. No action needed unless it affects most pages.",
	},
	"slow_host": {
		Title:       "Slow Server Response",
		Description: "Average TTFB for this host exceeds 5 seconds over the last 10 requests.",
		Impact:      "Slow server response degrades user experience and can negatively impact search rankings (Core Web Vitals).",
		Fix:         "Investigate server performance: check hosting, caching, database queries, and CDN configuration.",
	},
}
