package storage

import "testing"

func TestInsertAndGetFetch(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	urlID, err := db.UpsertURL(job.ID, "https://example.com/", "example.com", "pending", true, "seed")
	if err != nil {
		t.Fatalf("UpsertURL: %v", err)
	}

	input := FetchInput{
		JobID:               job.ID,
		FetchSeq:            1,
		RequestedURLID:      urlID,
		StatusCode:          200,
		RedirectHopCount:    0,
		TTFBMs:              150,
		ResponseBodySize:    4096,
		ContentType:         "text/html",
		ContentEncoding:     "gzip",
		ResponseHeadersJSON: `{"server":"nginx"}`,
		HTTPMethod:          "GET",
		FetchKind:           "full",
		RenderMode:          "static",
	}

	id, err := db.InsertFetch(input)
	if err != nil {
		t.Fatalf("InsertFetch: %v", err)
	}

	got, err := db.GetFetch(id)
	if err != nil {
		t.Fatalf("GetFetch: %v", err)
	}

	if got.JobID != job.ID {
		t.Errorf("expected job_id %q, got %q", job.ID, got.JobID)
	}
	if got.FetchSeq != 1 {
		t.Errorf("expected fetch_seq 1, got %d", got.FetchSeq)
	}
	if got.StatusCode.Int64 != 200 {
		t.Errorf("expected status_code 200, got %d", got.StatusCode.Int64)
	}
	if got.RenderMode != "static" {
		t.Errorf("expected render_mode %q, got %q", "static", got.RenderMode)
	}
	if got.HTTPMethod != "GET" {
		t.Errorf("expected http_method %q, got %q", "GET", got.HTTPMethod)
	}
	if got.FetchKind != "full" {
		t.Errorf("expected fetch_kind %q, got %q", "full", got.FetchKind)
	}
}

func TestInsertFetchWithOptionalFields(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	urlID, err := db.UpsertURL(job.ID, "https://example.com/page", "example.com", "pending", true, "crawl")
	if err != nil {
		t.Fatalf("UpsertURL: %v", err)
	}

	renderParams := `{"wait":5000}`
	fetchErr := "connection timeout"
	input := FetchInput{
		JobID:            job.ID,
		FetchSeq:         1,
		RequestedURLID:   urlID,
		StatusCode:       0,
		HTTPMethod:       "HEAD",
		FetchKind:        "headers_only",
		RenderMode:       "js",
		RenderParamsJSON: &renderParams,
		Error:            &fetchErr,
	}

	id, err := db.InsertFetch(input)
	if err != nil {
		t.Fatalf("InsertFetch: %v", err)
	}

	got, err := db.GetFetch(id)
	if err != nil {
		t.Fatalf("GetFetch: %v", err)
	}

	if got.HTTPMethod != "HEAD" {
		t.Errorf("expected http_method %q, got %q", "HEAD", got.HTTPMethod)
	}
	if got.FetchKind != "headers_only" {
		t.Errorf("expected fetch_kind %q, got %q", "headers_only", got.FetchKind)
	}
	if !got.RenderParamsJSON.Valid || got.RenderParamsJSON.String != renderParams {
		t.Errorf("expected render_params_json %q, got %v", renderParams, got.RenderParamsJSON)
	}
	if !got.Error.Valid || got.Error.String != fetchErr {
		t.Errorf("expected error %q, got %v", fetchErr, got.Error)
	}
}

func TestGetFetchByURL(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	urlID, err := db.UpsertURL(job.ID, "https://example.com/page", "example.com", "pending", true, "crawl")
	if err != nil {
		t.Fatalf("UpsertURL: %v", err)
	}

	input := FetchInput{
		JobID:          job.ID,
		FetchSeq:       1,
		RequestedURLID: urlID,
		StatusCode:     200,
		HTTPMethod:     "GET",
		FetchKind:      "full",
		RenderMode:     "static",
	}

	_, err = db.InsertFetch(input)
	if err != nil {
		t.Fatalf("InsertFetch: %v", err)
	}

	got, err := db.GetFetchByURL(job.ID, urlID)
	if err != nil {
		t.Fatalf("GetFetchByURL: %v", err)
	}

	if got.RequestedURLID != urlID {
		t.Errorf("expected requested_url_id %d, got %d", urlID, got.RequestedURLID)
	}
}
