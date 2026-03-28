package parser

// SchemaRule defines required and recommended properties for a Schema.org type.
type SchemaRule struct {
	Type        string
	Required    []string // Properties that MUST be present
	Recommended []string // Properties that SHOULD be present
}

// schemaRules maps Schema.org @type values to their validation rules.
var schemaRules = map[string]SchemaRule{
	"Article": {
		Required:    []string{"headline", "author", "datePublished"},
		Recommended: []string{"image", "dateModified", "publisher"},
	},
	"BlogPosting": {
		Required:    []string{"headline", "author", "datePublished"},
		Recommended: []string{"image", "dateModified", "publisher", "mainEntityOfPage"},
	},
	"NewsArticle": {
		Required:    []string{"headline", "author", "datePublished"},
		Recommended: []string{"image", "dateModified", "publisher"},
	},
	"Product": {
		Required:    []string{"name"},
		Recommended: []string{"image", "description", "offers", "brand", "sku"},
	},
	"Organization": {
		Required:    []string{"name"},
		Recommended: []string{"url", "logo", "description", "sameAs"},
	},
	"LocalBusiness": {
		Required:    []string{"name", "address"},
		Recommended: []string{"telephone", "openingHours", "geo", "image"},
	},
	"Person": {
		Required:    []string{"name"},
		Recommended: []string{"url", "image", "jobTitle"},
	},
	"WebSite": {
		Required:    []string{"name", "url"},
		Recommended: []string{"description", "publisher"},
	},
	"WebPage": {
		Required:    []string{"name"},
		Recommended: []string{"description", "url"},
	},
	"BreadcrumbList": {
		Required:    []string{"itemListElement"},
		Recommended: []string{},
	},
	"FAQPage": {
		Required:    []string{"mainEntity"},
		Recommended: []string{},
	},
	"HowTo": {
		Required:    []string{"name", "step"},
		Recommended: []string{"description", "image", "totalTime"},
	},
	"Event": {
		Required:    []string{"name", "startDate", "location"},
		Recommended: []string{"description", "endDate", "image", "offers"},
	},
	"Recipe": {
		Required:    []string{"name"},
		Recommended: []string{"image", "author", "description", "recipeIngredient", "recipeInstructions"},
	},
	"Review": {
		Required:    []string{"itemReviewed", "author"},
		Recommended: []string{"reviewRating", "datePublished"},
	},
	"Service": {
		Required:    []string{"name"},
		Recommended: []string{"description", "provider", "serviceType"},
	},
	"SoftwareApplication": {
		Required:    []string{"name"},
		Recommended: []string{"operatingSystem", "applicationCategory", "offers"},
	},
	"VideoObject": {
		Required:    []string{"name", "thumbnailUrl", "uploadDate"},
		Recommended: []string{"description", "contentUrl", "duration"},
	},
	"ImageObject": {
		Required:    []string{"contentUrl"},
		Recommended: []string{"name", "description"},
	},
	"ContactPage": {
		Required:    []string{"name"},
		Recommended: []string{"description", "url"},
	},
	"AboutPage": {
		Required:    []string{"name"},
		Recommended: []string{"description", "url"},
	},
	"CollectionPage": {
		Required:    []string{"name"},
		Recommended: []string{"description", "url", "mainEntity"},
	},
	"ItemList": {
		Required:    []string{"itemListElement"},
		Recommended: []string{"name", "numberOfItems"},
	},
	"Offer": {
		Required:    []string{"price", "priceCurrency"},
		Recommended: []string{"availability", "url", "seller"},
	},
	"AggregateRating": {
		Required:    []string{"ratingValue", "reviewCount"},
		Recommended: []string{"bestRating", "worstRating"},
	},
}
