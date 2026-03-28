// Package integration provides end-to-end crawl tests with a fixture site.
package integration

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
)

// NewFixtureSite creates an httptest.Server serving ~20 pages with various SEO
// scenarios for integration testing.
func NewFixtureSite() *httptest.Server {
	mux := http.NewServeMux()

	// robots.txt
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		host := r.Host
		base := scheme + "://" + host

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /admin/\nSitemap: %s/sitemap.xml\n", base)
	})

	// sitemap.xml — includes /hidden-page (orphan: not linked from any page)
	mux.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		// We'll inject the server URL at request time via Host header.
		scheme := "http"
		host := r.Host
		base := scheme + "://" + host

		base = html.EscapeString(base)

		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>%s/</loc></url>
  <url><loc>%s/about</loc></url>
  <url><loc>%s/blog</loc></url>
  <url><loc>%s/blog/post-1</loc></url>
  <url><loc>%s/blog/post-2</loc></url>
  <url><loc>%s/products</loc></url>
  <url><loc>%s/hidden-page</loc></url>
</urlset>`, base, base, base, base, base, base, base)
	})

	// ---- Homepage ----
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(404)
			fmt.Fprint(w, page404())
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Home Page - Test Site", "Welcome to the home page of our comprehensive test website",
			`<h1>Welcome to Our Test Website</h1>`+
				longParagraph("homepage")+
				`<nav>
				<a href="/about">About Us</a>
				<a href="/blog">Blog</a>
				<a href="/products">Products</a>
				<a href="/gallery">Gallery</a>
				<a href="/broken-link">Broken Link</a>
				<a href="/old-page">Old Page (Redirect)</a>
			</nav>`,
			"", // no extras
		))
	})

	// ---- About ----
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("About Us - Test Site", "Learn about our company and team and what drives us forward",
			`<h1>About Our Company</h1>`+
				longParagraph("about")+
				`<a href="/contact">Contact Us</a>`,
			"",
		))
	})

	// ---- Contact ----
	mux.HandleFunc("/contact", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Contact Us - Test Site", "Get in touch with our team through our contact page for inquiries",
			`<h1>Contact Us</h1>`+
				longParagraph("contact")+
				`<form><input type="text" name="name"><textarea name="message"></textarea></form>
				<a href="/">Back Home</a>`,
			"",
		))
	})

	// ---- Blog index ----
	mux.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Blog - Test Site", "Read our latest articles about technology and search engine optimization",
			`<h1>Blog</h1>`+
				longParagraph("blog-index")+
				`<ul>
				<li><a href="/blog/post-1">Post One: SEO Basics</a></li>
				<li><a href="/blog/post-2">Post Two: Advanced SEO</a></li>
				<li><a href="/blog/post-3">Post Three: Short</a></li>
				<li><a href="/blog/post-4">Post Four: Missing Meta</a></li>
				<li><a href="/blog/post-5">Post Five: Multiple H1</a></li>
			</ul>`,
			"",
		))
	})

	// ---- Blog Post 1: Has JSON-LD and hreflang ----
	mux.HandleFunc("/blog/post-1", func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		host := r.Host
		base := scheme + "://" + host

		safeBase := html.EscapeString(base)
		headExtra := `<script type="application/ld+json">
			{"@context":"https://schema.org","@type":"BlogPosting","headline":"SEO Basics Guide","author":{"@type":"Person","name":"Test Author"}}
			</script>
			<link rel="alternate" hreflang="es" href="` + safeBase + `/blog/post-1-es">
			<link rel="alternate" hreflang="en" href="` + safeBase + `/blog/post-1">`

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("SEO Basics Guide - Test Blog", "A comprehensive guide to search engine optimization basics for beginners",
			`<h1>SEO Basics Guide</h1>`+longParagraph("post-1")+`<a href="/blog">Back to Blog</a>`,
			headExtra,
		))
	})

	// ---- Blog Post 2: Duplicate title with post-1 ----
	mux.HandleFunc("/blog/post-2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Same title as post-1 to trigger duplicate_title in global issues
		fmt.Fprint(w, wrapHTML("SEO Basics Guide - Test Blog", "Another article about SEO fundamentals covering similar topics and techniques",
			`<h1>Advanced SEO Techniques</h1>`+longParagraph("post-2")+`<a href="/blog">Back to Blog</a>`,
			"",
		))
	})

	// ---- Blog Post 3: Thin content (very few words) ----
	mux.HandleFunc("/blog/post-3", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Short Post - Test Blog", "A brief post about nothing in particular just a few words here",
			`<h1>Short Post</h1><p>This is a very short post with minimal content.</p>`,
			"",
		))
	})

	// ---- Blog Post 4: Missing title and description ----
	mux.HandleFunc("/blog/post-4", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// No <title> and no meta description
		body := `<h1>Post Without Meta Tags</h1>` + longParagraph("post-4") + `<a href="/blog">Back to Blog</a>`
		fmt.Fprint(w, wrapHTML("", "", body, ""))
	})

	// ---- Blog Post 5: Multiple H1 tags ----
	mux.HandleFunc("/blog/post-5", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Multiple Headings Post - Test Blog", "A post that tests multiple H1 heading detection in the SEO crawler",
			`<h1>First H1 Heading</h1>`+longParagraph("post-5a")+
				`<h1>Second H1 Heading</h1>`+longParagraph("post-5b")+
				`<h1>Third H1 Heading</h1><p>Even a third one.</p>
			<a href="/blog">Back to Blog</a>`,
			"",
		))
	})

	// ---- Products index ----
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Products - Test Site", "Browse our complete product catalog featuring widgets and accessories",
			`<h1>Our Products</h1>`+longParagraph("products")+
				`<ul>
				<li><a href="/products/widget">Basic Widget</a></li>
				<li><a href="/products/widget-pro">Widget Pro</a></li>
			</ul>`,
			"",
		))
	})

	// ---- Product Widget: canonical points to widget-pro ----
	mux.HandleFunc("/products/widget", func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		host := r.Host
		base := scheme + "://" + host

		safeBase := html.EscapeString(base)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Basic Widget - Test Site", "The basic widget product page with all the details you need to know",
			`<h1>Basic Widget</h1>`+longParagraph("widget")+`<a href="/products">All Products</a>`,
			`<link rel="canonical" href="`+safeBase+`/products/widget-pro">`,
		))
	})

	// ---- Product Widget Pro: self-canonical ----
	mux.HandleFunc("/products/widget-pro", func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		host := r.Host
		base := scheme + "://" + host

		safeBase := html.EscapeString(base)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Widget Pro - Test Site", "The professional widget with advanced features and premium quality materials",
			`<h1>Widget Pro</h1>`+longParagraph("widget-pro")+`<a href="/products">All Products</a>`,
			`<link rel="canonical" href="`+safeBase+`/products/widget-pro">`,
		))
	})

	// ---- Redirect chain: /old-page → 301 → /redirect-step → 302 → /about ----
	mux.HandleFunc("/old-page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/redirect-step")
		w.WriteHeader(301)
	})
	mux.HandleFunc("/redirect-step", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/about")
		w.WriteHeader(302)
	})

	// ---- 404 page ----
	mux.HandleFunc("/broken-link", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, page404())
	})

	// ---- Gallery: images without alt ----
	mux.HandleFunc("/gallery", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Gallery - Test Site", "Browse our image gallery showcasing various photos and artwork pieces",
			`<h1>Image Gallery</h1>`+longParagraph("gallery")+
				`<img src="/images/photo1.jpg">
			<img src="/images/photo2.png">
			<img src="/images/photo3.gif">
			<a href="/">Back Home</a>`,
			"",
		))
	})

	// ---- Hidden page (orphan: only in sitemap, no inbound links) ----
	mux.HandleFunc("/hidden-page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, wrapHTML("Hidden Page - Test Site", "This is a hidden page that is only discoverable through the sitemap file",
			`<h1>Hidden Page</h1>`+longParagraph("hidden")+`<a href="/">Home</a>`,
			"",
		))
	})

	return httptest.NewServer(mux)
}

// wrapHTML builds a complete HTML document.
func wrapHTML(title, description, body, headExtra string) string {
	var sb strings.Builder
	safeTitle := html.EscapeString(title)
	safeDescription := html.EscapeString(description)
	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">`)
	if safeTitle != "" {
		sb.WriteString("\n<title>")
		sb.WriteString(safeTitle)
		sb.WriteString("</title>")
	}
	if safeDescription != "" {
		sb.WriteString(`
<meta name="description" content="`)
		sb.WriteString(safeDescription)
		sb.WriteString(`">`)
	}
	if headExtra != "" {
		sb.WriteString("\n")
		sb.WriteString(headExtra)
	}
	sb.WriteString(`
</head>
<body>
`)
	sb.WriteString(body)
	sb.WriteString(`
</body>
</html>`)
	return sb.String()
}

// longParagraph generates >200 words of deterministic filler for a given seed.
func longParagraph(seed string) string {
	// Each sentence has ~15 words; 15 sentences ≈ 225 words.
	sentences := []string{
		"The importance of search engine optimization cannot be overstated in the modern digital landscape today.",
		"Website owners must carefully consider their content strategy to attract and retain organic search traffic.",
		"Technical SEO involves optimizing the underlying code and structure of a website for better indexing.",
		"Content quality remains one of the most critical ranking factors according to all major search engines.",
		"Mobile responsiveness has become essential as more users access websites from smartphones and tablet devices.",
		"Page loading speed directly impacts user experience and is a confirmed ranking signal for search results.",
		"Internal linking helps distribute page authority throughout a website and improves overall crawlability significantly.",
		"Meta descriptions serve as advertising copy that can influence click through rates from search result pages.",
		"Heading tags provide hierarchical structure to content helping both users and search engines understand the page.",
		"Image optimization includes proper sizing compression and descriptive alternative text for accessibility and SEO benefits.",
		"Schema markup provides search engines with explicit clues about the meaning of page content and structure.",
		"Regular content audits help identify outdated thin or duplicate content that may be hurting overall performance.",
		"Backlink profiles remain a strong ranking factor but quality matters much more than sheer quantity of links.",
		"User engagement metrics like bounce rate and dwell time can indirectly influence search engine ranking positions.",
		"Keeping up with algorithm updates and industry best practices is essential for maintaining search visibility over time.",
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<p>Section about %s. ", seed))
	for _, s := range sentences {
		sb.WriteString(s)
		sb.WriteString(" ")
	}
	sb.WriteString("</p>")
	return sb.String()
}

// page404 returns a minimal 404 page body.
func page404() string {
	return `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>404 Not Found</title></head>
<body><h1>Page Not Found</h1><p>The requested page could not be found.</p></body>
</html>`
}
