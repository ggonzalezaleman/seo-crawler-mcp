package storage

import (
	"fmt"
)

// eventColumns is the canonical SELECT list for crawl_events.
const eventColumns = `id, job_id, timestamp, event_type, details_json, url`

// scanEvent scans a row into a CrawlEvent using the eventColumns order.
func scanEvent(sc interface{ Scan(...any) error }) (CrawlEvent, error) {
	var e CrawlEvent
	err := sc.Scan(
		&e.ID, &e.JobID, &e.Timestamp, &e.EventType,
		&e.DetailsJSON, &e.URL,
	)
	return e, err
}

// InsertEvent creates a new crawl event and returns its ID.
func (db *DB) InsertEvent(jobID, eventType string, detailsJSON, url *string) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO crawl_events (job_id, event_type, details_json, url)
		 VALUES (?, ?, ?, ?)`,
		jobID, eventType, detailsJSON, url,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting event %q for job %q: %w", eventType, jobID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert ID for event: %w", err)
	}

	return id, nil
}

// GetEventsByJob returns the most recent events for a job, ordered by timestamp desc.
func (db *DB) GetEventsByJob(jobID string, limit int) ([]CrawlEvent, error) {
	rows, err := db.Query(
		`SELECT `+eventColumns+` FROM crawl_events
		 WHERE job_id = ?
		 ORDER BY id DESC LIMIT ?`,
		jobID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying events for job %q: %w", jobID, err)
	}
	defer rows.Close()

	events := []CrawlEvent{}
	for rows.Next() {
		e, scanErr := scanEvent(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning event row: %w", scanErr)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating event rows: %w", err)
	}

	return events, nil
}
