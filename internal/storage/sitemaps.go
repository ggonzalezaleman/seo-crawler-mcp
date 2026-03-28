package storage

import (
	"fmt"
)

// SitemapEntryInput holds parameters for InsertSitemapEntry.
type SitemapEntryInput struct {
	JobID            string
	URL              string
	SourceSitemapURL string
	SourceHost       string
	Lastmod          *string
	Changefreq       *string
	Priority         *float64
}

// sitemapColumns is the canonical SELECT list for sitemap_entries.
const sitemapColumns = `id, job_id, url, source_sitemap_url, source_host,
	lastmod, changefreq, priority, reconciliation_status`

// scanSitemapEntry scans a row into a SitemapEntry.
func scanSitemapEntry(sc interface{ Scan(...any) error }) (SitemapEntry, error) {
	var s SitemapEntry
	err := sc.Scan(
		&s.ID, &s.JobID, &s.URL, &s.SourceSitemapURL, &s.SourceHost,
		&s.Lastmod, &s.Changefreq, &s.Priority, &s.ReconciliationStatus,
	)
	return s, err
}

// InsertSitemapEntry creates a new sitemap entry and returns its ID.
func (db *DB) InsertSitemapEntry(input SitemapEntryInput) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO sitemap_entries (job_id, url, source_sitemap_url, source_host,
			lastmod, changefreq, priority)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.JobID, input.URL, input.SourceSitemapURL, input.SourceHost,
		input.Lastmod, input.Changefreq, input.Priority,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting sitemap entry %q for job %q: %w", input.URL, input.JobID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert ID for sitemap entry: %w", err)
	}

	return id, nil
}

// GetSitemapEntriesByJob returns sitemap entries for a job, up to limit rows.
func (db *DB) GetSitemapEntriesByJob(jobID string, limit int) ([]SitemapEntry, error) {
	rows, err := db.Query(
		`SELECT `+sitemapColumns+` FROM sitemap_entries
		 WHERE job_id = ? ORDER BY id ASC LIMIT ?`,
		jobID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying sitemap entries for job %q: %w", jobID, err)
	}
	defer rows.Close()

	entries := []SitemapEntry{}
	for rows.Next() {
		s, scanErr := scanSitemapEntry(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning sitemap entry row: %w", scanErr)
		}
		entries = append(entries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating sitemap entry rows: %w", err)
	}

	return entries, nil
}

// UpdateSitemapReconciliation updates the reconciliation_status of a sitemap entry.
func (db *DB) UpdateSitemapReconciliation(id int64, status string) error {
	result, err := db.Exec(
		`UPDATE sitemap_entries SET reconciliation_status = ? WHERE id = ?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("updating reconciliation status for sitemap entry %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for sitemap entry %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("sitemap entry %d not found", id)
	}

	return nil
}
