package storage

import (
	"database/sql"
	"fmt"
)

// LlmsFindingInput holds parameters for UpsertLlmsFinding.
type LlmsFindingInput struct {
	JobID              string
	Host               string
	Present            bool
	RawContent         *string
	SectionsJSON       *string
	ReferencedURLsJSON *string
}

// llmsColumns is the canonical SELECT list for llms_findings.
const llmsColumns = `id, job_id, host, present, raw_content, sections_json, referenced_urls_json`

// scanLlmsFinding scans a row into a LlmsFinding.
func scanLlmsFinding(sc interface{ Scan(...any) error }) (LlmsFinding, error) {
	var l LlmsFinding
	var present int
	err := sc.Scan(
		&l.ID, &l.JobID, &l.Host, &present,
		&l.RawContent, &l.SectionsJSON, &l.ReferencedURLsJSON,
	)
	l.Present = present == 1
	return l, err
}

// UpsertLlmsFinding inserts or updates an llms.txt finding for a host.
func (db *DB) UpsertLlmsFinding(input LlmsFindingInput) (int64, error) {
	_, err := db.Exec(
		`INSERT INTO llms_findings (job_id, host, present, raw_content, sections_json, referenced_urls_json)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(job_id, host) DO UPDATE SET
			present = excluded.present,
			raw_content = excluded.raw_content,
			sections_json = excluded.sections_json,
			referenced_urls_json = excluded.referenced_urls_json`,
		input.JobID, input.Host, boolToInt(input.Present),
		input.RawContent, input.SectionsJSON, input.ReferencedURLsJSON,
	)
	if err != nil {
		return 0, fmt.Errorf("upserting llms finding for host %q in job %q: %w", input.Host, input.JobID, err)
	}

	var id int64
	err = db.QueryRow(
		"SELECT id FROM llms_findings WHERE job_id = ? AND host = ?",
		input.JobID, input.Host,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("getting ID for llms finding host %q in job %q: %w", input.Host, input.JobID, err)
	}

	return id, nil
}

// GetLlmsFindingByHost retrieves an llms.txt finding by job ID and host.
func (db *DB) GetLlmsFindingByHost(jobID, host string) (*LlmsFinding, error) {
	row := db.QueryRow(
		`SELECT `+llmsColumns+` FROM llms_findings
		 WHERE job_id = ? AND host = ?`,
		jobID, host,
	)

	l, err := scanLlmsFinding(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("llms finding for host %q in job %q not found", host, jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("scanning llms finding for host %q in job %q: %w", host, jobID, err)
	}

	return &l, nil
}
