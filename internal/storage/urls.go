package storage

import (
	"database/sql"
	"fmt"
)

// boolToInt converts a bool to 0/1 for SQLite INTEGER columns.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// UpsertURL inserts a URL if it doesn't exist, and returns its ID either way.
func (db *DB) UpsertURL(
	jobID, normalizedURL, host, status string,
	isInternal bool,
	discoveredVia string,
) (int64, error) {
	_, err := db.Exec(
		`INSERT OR IGNORE INTO urls (job_id, normalized_url, host, status, is_internal, discovered_via)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		jobID, normalizedURL, host, status, boolToInt(isInternal), discoveredVia,
	)
	if err != nil {
		return 0, fmt.Errorf("upserting URL %q for job %q: %w", normalizedURL, jobID, err)
	}

	var id int64
	err = db.QueryRow(
		`SELECT id FROM urls WHERE job_id = ? AND normalized_url = ?`,
		jobID, normalizedURL,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("fetching ID for URL %q in job %q: %w", normalizedURL, jobID, err)
	}

	return id, nil
}

// GetURL retrieves a URL by its auto-increment ID.
func (db *DB) GetURL(id int64) (*URL, error) {
	row := db.QueryRow(
		`SELECT id, job_id, normalized_url, host, status, is_internal, discovered_via, created_at
		 FROM urls WHERE id = ?`,
		id,
	)

	var u URL
	var isInternal int
	err := row.Scan(
		&u.ID, &u.JobID, &u.NormalizedURL, &u.Host,
		&u.Status, &isInternal, &u.DiscoveredVia, &u.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("URL with id %d not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("scanning URL %d: %w", id, err)
	}
	u.IsInternal = isInternal == 1

	return &u, nil
}

// GetURLByNormalized retrieves a URL by job ID and normalized URL.
func (db *DB) GetURLByNormalized(jobID, normalizedURL string) (*URL, error) {
	row := db.QueryRow(
		`SELECT id, job_id, normalized_url, host, status, is_internal, discovered_via, created_at
		 FROM urls WHERE job_id = ? AND normalized_url = ?`,
		jobID, normalizedURL,
	)

	var u URL
	var isInternal int
	err := row.Scan(
		&u.ID, &u.JobID, &u.NormalizedURL, &u.Host,
		&u.Status, &isInternal, &u.DiscoveredVia, &u.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("URL %q not found in job %q", normalizedURL, jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("scanning URL %q in job %q: %w", normalizedURL, jobID, err)
	}
	u.IsInternal = isInternal == 1

	return &u, nil
}

// UpdateURLStatus changes the status of a URL by ID.
func (db *DB) UpdateURLStatus(id int64, status string) error {
	result, err := db.Exec(
		`UPDATE urls SET status = ? WHERE id = ?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("updating status for URL %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for URL %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("URL with id %d not found", id)
	}

	return nil
}

// CountURLsByStatus returns a map of status → count for all URLs in a job.
func (db *DB) CountURLsByStatus(jobID string) (map[string]int, error) {
	rows, err := db.Query(
		`SELECT status, COUNT(*) FROM urls WHERE job_id = ? GROUP BY status`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("counting URLs by status for job %q: %w", jobID, err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scanning URL status count: %w", err)
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating URL status counts: %w", err)
	}

	return counts, nil
}
