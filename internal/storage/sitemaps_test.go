package storage

import "testing"

func TestInsertAndGetSitemapEntries(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	lastmod := "2026-01-15"
	priority := 0.8
	id, err := db.InsertSitemapEntry(SitemapEntryInput{
		JobID:            job.ID,
		URL:              "https://example.com/page1",
		SourceSitemapURL: "https://example.com/sitemap.xml",
		SourceHost:       "example.com",
		Lastmod:          &lastmod,
		Priority:         &priority,
	})
	if err != nil {
		t.Fatalf("InsertSitemapEntry: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero sitemap entry ID")
	}

	_, err = db.InsertSitemapEntry(SitemapEntryInput{
		JobID:            job.ID,
		URL:              "https://example.com/page2",
		SourceSitemapURL: "https://example.com/sitemap.xml",
		SourceHost:       "example.com",
	})
	if err != nil {
		t.Fatalf("InsertSitemapEntry 2: %v", err)
	}

	entries, err := db.GetSitemapEntriesByJob(job.ID, 1000)
	if err != nil {
		t.Fatalf("GetSitemapEntriesByJob: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].URL != "https://example.com/page1" {
		t.Errorf("expected URL %q, got %q", "https://example.com/page1", entries[0].URL)
	}
	if !entries[0].Priority.Valid || entries[0].Priority.Float64 != 0.8 {
		t.Errorf("expected priority 0.8, got %v", entries[0].Priority)
	}
}

func TestUpdateSitemapReconciliation(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	id, err := db.InsertSitemapEntry(SitemapEntryInput{
		JobID:            job.ID,
		URL:              "https://example.com/page",
		SourceSitemapURL: "https://example.com/sitemap.xml",
		SourceHost:       "example.com",
	})
	if err != nil {
		t.Fatalf("InsertSitemapEntry: %v", err)
	}

	err = db.UpdateSitemapReconciliation(id, "matched")
	if err != nil {
		t.Fatalf("UpdateSitemapReconciliation: %v", err)
	}

	entries, err := db.GetSitemapEntriesByJob(job.ID, 1000)
	if err != nil {
		t.Fatalf("GetSitemapEntriesByJob: %v", err)
	}
	if entries[0].ReconciliationStatus != "matched" {
		t.Errorf("expected status %q, got %q", "matched", entries[0].ReconciliationStatus)
	}
}
