package storage

import "testing"

func TestInsertAndGetEdges(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	srcID, err := db.UpsertURL(job.ID, "https://example.com/", "example.com", "pending", true, "seed")
	if err != nil {
		t.Fatalf("UpsertURL src: %v", err)
	}

	tgtID, err := db.UpsertURL(job.ID, "https://example.com/about", "example.com", "pending", true, "crawl")
	if err != nil {
		t.Fatalf("UpsertURL tgt: %v", err)
	}

	anchor := "About Us"
	input := EdgeInput{
		JobID:                 job.ID,
		SourceURLID:           srcID,
		NormalizedTargetURLID: tgtID,
		SourceKind:            "html",
		RelationType:          "hyperlink",
		DiscoveryMode:         "crawl",
		AnchorText:            &anchor,
		IsInternal:            true,
		DeclaredTargetURL:     "https://example.com/about",
	}

	id, err := db.InsertEdge(input)
	if err != nil {
		t.Fatalf("InsertEdge: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero edge ID")
	}

	edges, err := db.GetEdgesBySource(job.ID, srcID, 10, "")
	if err != nil {
		t.Fatalf("GetEdgesBySource: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if !edges[0].AnchorText.Valid || edges[0].AnchorText.String != "About Us" {
		t.Errorf("expected anchor_text %q, got %v", "About Us", edges[0].AnchorText)
	}

	edges, err = db.GetEdgesByTarget(job.ID, tgtID, 10, "")
	if err != nil {
		t.Fatalf("GetEdgesByTarget: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge by target, got %d", len(edges))
	}
}

func TestCountEdges(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	srcID, err := db.UpsertURL(job.ID, "https://example.com/", "example.com", "pending", true, "seed")
	if err != nil {
		t.Fatalf("UpsertURL: %v", err)
	}

	tgtID, err := db.UpsertURL(job.ID, "https://example.com/a", "example.com", "pending", true, "crawl")
	if err != nil {
		t.Fatalf("UpsertURL: %v", err)
	}

	for i := range 3 {
		tgt2, err := db.UpsertURL(
			job.ID,
			"https://example.com/p"+string(rune('0'+i)),
			"example.com", "pending", true, "crawl",
		)
		if err != nil {
			t.Fatalf("UpsertURL %d: %v", i, err)
		}
		_ = tgt2
		_, err = db.InsertEdge(EdgeInput{
			JobID:                 job.ID,
			SourceURLID:           srcID,
			NormalizedTargetURLID: tgtID,
			SourceKind:            "html",
			RelationType:          "hyperlink",
			DiscoveryMode:         "crawl",
			IsInternal:            true,
			DeclaredTargetURL:     "https://example.com/a",
		})
		if err != nil {
			t.Fatalf("InsertEdge %d: %v", i, err)
		}
	}

	count, err := db.CountEdges(job.ID)
	if err != nil {
		t.Fatalf("CountEdges: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 edges, got %d", count)
	}
}
