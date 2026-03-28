package storage

import "testing"

func TestUpsertAndGetLlmsFinding(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	raw := "# llms.txt\nAllow: /"
	sections := `[{"title":"root","content":"Allow: /"}]`
	urls := `["https://example.com/"]`

	id, err := db.UpsertLlmsFinding(LlmsFindingInput{
		JobID:              job.ID,
		Host:               "example.com",
		Present:            true,
		RawContent:         &raw,
		SectionsJSON:       &sections,
		ReferencedURLsJSON: &urls,
	})
	if err != nil {
		t.Fatalf("UpsertLlmsFinding: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero llms finding ID")
	}

	got, err := db.GetLlmsFindingByHost(job.ID, "example.com")
	if err != nil {
		t.Fatalf("GetLlmsFindingByHost: %v", err)
	}

	if !got.Present {
		t.Error("expected present=true")
	}
	if !got.RawContent.Valid || got.RawContent.String != raw {
		t.Errorf("expected raw_content %q, got %v", raw, got.RawContent)
	}

	// Test upsert (update existing).
	raw2 := "# updated"
	id2, err := db.UpsertLlmsFinding(LlmsFindingInput{
		JobID:      job.ID,
		Host:       "example.com",
		Present:    false,
		RawContent: &raw2,
	})
	if err != nil {
		t.Fatalf("UpsertLlmsFinding update: %v", err)
	}
	if id2 != id {
		t.Errorf("expected same ID %d after upsert, got %d", id, id2)
	}

	got2, err := db.GetLlmsFindingByHost(job.ID, "example.com")
	if err != nil {
		t.Fatalf("GetLlmsFindingByHost after update: %v", err)
	}
	if got2.Present {
		t.Error("expected present=false after update")
	}
	if !got2.RawContent.Valid || got2.RawContent.String != "# updated" {
		t.Errorf("expected updated raw_content, got %v", got2.RawContent)
	}
}

func TestGetLlmsFindingNotFound(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	_, err = db.GetLlmsFindingByHost(job.ID, "nonexistent.com")
	if err == nil {
		t.Fatal("expected error for nonexistent host, got nil")
	}
}
