package storage

import "testing"

func TestInsertAndGetAssets(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	urlID, err := db.UpsertURL(job.ID, "https://example.com/style.css", "example.com", "pending", true, "crawl")
	if err != nil {
		t.Fatalf("UpsertURL: %v", err)
	}

	ct := "text/css"
	sc := 200
	cl := int64(12345)
	id, err := db.InsertAsset(AssetInput{
		JobID:         job.ID,
		URLID:         urlID,
		ContentType:   &ct,
		StatusCode:    &sc,
		ContentLength: &cl,
	})
	if err != nil {
		t.Fatalf("InsertAsset: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero asset ID")
	}

	assets, err := db.GetAssetsByJob(job.ID, 1000)
	if err != nil {
		t.Fatalf("GetAssetsByJob: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	if !assets[0].ContentType.Valid || assets[0].ContentType.String != "text/css" {
		t.Errorf("expected content_type %q, got %v", "text/css", assets[0].ContentType)
	}
}

func TestInsertAssetReference(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	assetURLID, err := db.UpsertURL(job.ID, "https://example.com/app.js", "example.com", "pending", true, "crawl")
	if err != nil {
		t.Fatalf("UpsertURL asset: %v", err)
	}

	pageURLID, err := db.UpsertURL(job.ID, "https://example.com/", "example.com", "pending", true, "seed")
	if err != nil {
		t.Fatalf("UpsertURL page: %v", err)
	}

	id, err := db.InsertAssetReference(job.ID, assetURLID, pageURLID, "script")
	if err != nil {
		t.Fatalf("InsertAssetReference: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero asset reference ID")
	}
}
