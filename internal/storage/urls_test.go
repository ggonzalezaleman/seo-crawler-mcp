package storage

import "testing"

func TestUpsertURL(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("full", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	// Insert new URL.
	id1, err := db.UpsertURL(job.ID, "https://example.com/", "example.com", "pending", true, "seed")
	if err != nil {
		t.Fatalf("UpsertURL (new): %v", err)
	}
	if id1 == 0 {
		t.Fatal("expected non-zero ID for new URL")
	}

	// Duplicate insert should return same ID.
	id2, err := db.UpsertURL(job.ID, "https://example.com/", "example.com", "pending", true, "seed")
	if err != nil {
		t.Fatalf("UpsertURL (duplicate): %v", err)
	}
	if id2 != id1 {
		t.Errorf("duplicate insert: expected ID %d, got %d", id1, id2)
	}

	// Different URL should get a different ID.
	id3, err := db.UpsertURL(job.ID, "https://example.com/about", "example.com", "pending", true, "crawl")
	if err != nil {
		t.Fatalf("UpsertURL (different): %v", err)
	}
	if id3 == id1 {
		t.Errorf("different URL got same ID %d", id1)
	}

	// Verify via GetURL.
	u, err := db.GetURL(id1)
	if err != nil {
		t.Fatalf("GetURL: %v", err)
	}
	if u.NormalizedURL != "https://example.com/" {
		t.Errorf("unexpected URL: %q", u.NormalizedURL)
	}
	if !u.IsInternal {
		t.Error("expected IsInternal=true")
	}

	// Verify via GetURLByNormalized.
	u2, err := db.GetURLByNormalized(job.ID, "https://example.com/about")
	if err != nil {
		t.Fatalf("GetURLByNormalized: %v", err)
	}
	if u2.ID != id3 {
		t.Errorf("expected ID %d, got %d", id3, u2.ID)
	}
}

func TestCountURLsByStatus(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("full", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	// Insert URLs with mixed statuses.
	urls := []struct {
		url    string
		status string
	}{
		{"https://example.com/1", "pending"},
		{"https://example.com/2", "pending"},
		{"https://example.com/3", "crawled"},
		{"https://example.com/4", "error"},
	}

	for _, u := range urls {
		_, err := db.UpsertURL(job.ID, u.url, "example.com", u.status, true, "seed")
		if err != nil {
			t.Fatalf("UpsertURL %q: %v", u.url, err)
		}
	}

	counts, err := db.CountURLsByStatus(job.ID)
	if err != nil {
		t.Fatalf("CountURLsByStatus: %v", err)
	}

	if counts["pending"] != 2 {
		t.Errorf("expected 2 pending, got %d", counts["pending"])
	}
	if counts["crawled"] != 1 {
		t.Errorf("expected 1 crawled, got %d", counts["crawled"])
	}
	if counts["error"] != 1 {
		t.Errorf("expected 1 error, got %d", counts["error"])
	}
}
