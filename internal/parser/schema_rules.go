package parser

// SchemaRule defines required and recommended properties for a Schema.org type.
type SchemaRule struct {
	Type         string
	Required     []string // Properties Google requires for rich results (Tier 1)
	Recommended  []string // Properties that are best practice (Tier 2)
	Source       string   // "google_rich_results" or "schema_org_best_practice"
	GoogleDocURL string   // URL to Google's structured data doc (empty if no rich result)
}

// schemaRules maps Schema.org @type values to their validation rules.
var schemaRules = map[string]SchemaRule{
	// ── Google Rich Result Types (Tier 1) ──────────────────────────────
	"Article": {
		Required:     []string{"headline", "author", "datePublished", "image"},
		Recommended:  []string{"dateModified", "publisher", "mainEntityOfPage"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/article",
	},
	"BlogPosting": {
		Required:     []string{"headline", "author", "datePublished", "image"},
		Recommended:  []string{"dateModified", "publisher", "mainEntityOfPage"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/article",
	},
	"NewsArticle": {
		Required:     []string{"headline", "author", "datePublished", "image"},
		Recommended:  []string{"dateModified", "publisher"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/article",
	},
	"Product": {
		Required:     []string{"name"},
		Recommended:  []string{"image", "description", "offers", "brand", "sku", "review", "aggregateRating"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/product",
	},
	"LocalBusiness": {
		Required:     []string{"name", "address"},
		Recommended:  []string{"telephone", "openingHours", "geo", "image", "url", "priceRange"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/local-business",
	},
	"BreadcrumbList": {
		Required:     []string{"itemListElement"},
		Recommended:  []string{},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/breadcrumb",
	},
	"FAQPage": {
		Required:     []string{"mainEntity"},
		Recommended:  []string{},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/faqpage",
	},
	"HowTo": {
		Required:     []string{"name", "step"},
		Recommended:  []string{"description", "image", "totalTime"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/how-to",
	},
	"Event": {
		Required:     []string{"name", "startDate", "location"},
		Recommended:  []string{"description", "endDate", "image", "offers", "organizer", "performer"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/event",
	},
	"Recipe": {
		Required:     []string{"name", "image"},
		Recommended:  []string{"author", "description", "recipeIngredient", "recipeInstructions", "prepTime", "cookTime", "totalTime", "nutrition"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/recipe",
	},
	"Review": {
		Required:     []string{"itemReviewed", "author", "reviewRating"},
		Recommended:  []string{"datePublished", "reviewBody"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/review-snippet",
	},
	"SoftwareApplication": {
		Required:     []string{"name", "offers"},
		Recommended:  []string{"operatingSystem", "applicationCategory", "aggregateRating", "review"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/software-app",
	},
	"VideoObject": {
		Required:     []string{"name", "thumbnailUrl", "uploadDate"},
		Recommended:  []string{"description", "contentUrl", "duration", "embedUrl"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/video",
	},
	"Offer": {
		Required:     []string{"price", "priceCurrency"},
		Recommended:  []string{"availability", "url", "seller", "validFrom"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/product",
	},
	"AggregateRating": {
		Required:     []string{"ratingValue"},
		Recommended:  []string{"reviewCount", "ratingCount", "bestRating", "worstRating"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/review-snippet",
	},
	"ItemList": {
		Required:     []string{"itemListElement"},
		Recommended:  []string{"name", "numberOfItems"},
		Source:       "google_rich_results",
		GoogleDocURL: "https://developers.google.com/search/docs/appearance/structured-data/carousel",
	},

	// ── Schema.org Best Practice Types (Tier 2) ────────────────────────
	"Organization": {
		Required:    []string{},
		Recommended: []string{"name", "url", "logo", "description", "sameAs", "contactPoint"},
		Source:      "schema_org_best_practice",
	},
	"Person": {
		Required:    []string{},
		Recommended: []string{"name", "url", "image", "jobTitle", "sameAs"},
		Source:      "schema_org_best_practice",
	},
	"WebSite": {
		Required:    []string{},
		Recommended: []string{"name", "url", "description", "publisher", "potentialAction"},
		Source:      "schema_org_best_practice",
	},
	"WebPage": {
		Required:    []string{},
		Recommended: []string{"name", "description", "url", "breadcrumb"},
		Source:      "schema_org_best_practice",
	},
	"Service": {
		Required:    []string{},
		Recommended: []string{"name", "description", "provider", "serviceType", "areaServed"},
		Source:      "schema_org_best_practice",
	},
	"ContactPage": {
		Required:    []string{},
		Recommended: []string{"name", "description", "url"},
		Source:      "schema_org_best_practice",
	},
	"AboutPage": {
		Required:    []string{},
		Recommended: []string{"name", "description", "url"},
		Source:      "schema_org_best_practice",
	},
	"CollectionPage": {
		Required:    []string{},
		Recommended: []string{"name", "description", "url", "mainEntity"},
		Source:      "schema_org_best_practice",
	},
	"ImageObject": {
		Required:    []string{},
		Recommended: []string{"contentUrl", "name", "description", "caption"},
		Source:      "schema_org_best_practice",
	},
}
