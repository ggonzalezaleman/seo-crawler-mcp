package storage

import (
	"database/sql"
	"fmt"
)

// FetchInput holds parameters for InsertFetch.
type FetchInput struct {
	JobID               string
	FetchSeq            int
	RequestedURLID      int64
	FinalURLID          *int64
	StatusCode          int
	RedirectHopCount    int
	TTFBMs              int64
	ResponseBodySize    int64
	ContentType         string
	ContentEncoding     string
	ResponseHeadersJSON string
	HTTPMethod          string  // defaults to "GET" in schema, caller can set "HEAD"
	FetchKind           string  // defaults to "full", caller can set "headers_only"
	RenderMode          string
	RenderParamsJSON    *string // nullable
	Error               *string // nullable
}

// fetchColumns is the canonical SELECT list for fetches.
const fetchColumns = `id, job_id, fetch_seq, requested_url_id, final_url_id,
	status_code, redirect_hop_count, ttfb_ms, response_body_size,
	content_type, content_encoding, response_headers_json,
	http_method, fetch_kind, render_mode, render_params_json, fetched_at, error`

// scanFetch scans a row into a Fetch using the fetchColumns order.
func scanFetch(sc interface{ Scan(...any) error }) (Fetch, error) {
	var f Fetch
	err := sc.Scan(
		&f.ID, &f.JobID, &f.FetchSeq, &f.RequestedURLID, &f.FinalURLID,
		&f.StatusCode, &f.RedirectHopCount, &f.TTFBMS, &f.ResponseBodySize,
		&f.ContentType, &f.ContentEncoding, &f.ResponseHeadersJSON,
		&f.HTTPMethod, &f.FetchKind, &f.RenderMode, &f.RenderParamsJSON,
		&f.FetchedAt, &f.Error,
	)
	return f, err
}

// InsertFetch creates a new fetch record and returns its ID.
func (db *DB) InsertFetch(input FetchInput) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO fetches (job_id, fetch_seq, requested_url_id, final_url_id,
			status_code, redirect_hop_count, ttfb_ms, response_body_size,
			content_type, content_encoding, response_headers_json,
			http_method, fetch_kind, render_mode, render_params_json, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.JobID, input.FetchSeq, input.RequestedURLID, input.FinalURLID,
		input.StatusCode, input.RedirectHopCount, input.TTFBMs, input.ResponseBodySize,
		input.ContentType, input.ContentEncoding, input.ResponseHeadersJSON,
		input.HTTPMethod, input.FetchKind, input.RenderMode, input.RenderParamsJSON, input.Error,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting fetch for job %q seq %d: %w", input.JobID, input.FetchSeq, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert ID for fetch: %w", err)
	}

	return id, nil
}

// GetFetch retrieves a fetch by its auto-increment ID.
func (db *DB) GetFetch(id int64) (*Fetch, error) {
	row := db.QueryRow(
		`SELECT `+fetchColumns+` FROM fetches WHERE id = ?`, id,
	)

	f, err := scanFetch(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fetch %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("scanning fetch %d: %w", id, err)
	}

	return &f, nil
}

// GetFetchByURL retrieves a fetch by job ID and requested URL ID.
func (db *DB) GetFetchByURL(jobID string, urlID int64) (*Fetch, error) {
	row := db.QueryRow(
		`SELECT `+fetchColumns+` FROM fetches
		 WHERE job_id = ? AND requested_url_id = ?
		 ORDER BY fetch_seq DESC LIMIT 1`,
		jobID, urlID,
	)

	f, err := scanFetch(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fetch for URL %d in job %q not found", urlID, jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("scanning fetch for URL %d in job %q: %w", urlID, jobID, err)
	}

	return &f, nil
}
