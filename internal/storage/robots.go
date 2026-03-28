package storage

import (
	"fmt"
)

// RobotsDirectiveInput holds parameters for InsertRobotsDirective.
type RobotsDirectiveInput struct {
	JobID       string
	Host        string
	UserAgent   string
	RuleType    string
	PathPattern string
	SourceURL   string
}

// robotsColumns is the canonical SELECT list for robots_directives.
const robotsColumns = `id, job_id, host, user_agent, rule_type, path_pattern, source_url`

// scanRobotsDirective scans a row into a RobotsDirective.
func scanRobotsDirective(sc interface{ Scan(...any) error }) (RobotsDirective, error) {
	var r RobotsDirective
	err := sc.Scan(
		&r.ID, &r.JobID, &r.Host, &r.UserAgent,
		&r.RuleType, &r.PathPattern, &r.SourceURL,
	)
	return r, err
}

// InsertRobotsDirective creates a new robots directive and returns its ID.
func (db *DB) InsertRobotsDirective(input RobotsDirectiveInput) (int64, error) {
	result, err := db.Exec(
		`INSERT INTO robots_directives (job_id, host, user_agent, rule_type, path_pattern, source_url)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		input.JobID, input.Host, input.UserAgent, input.RuleType, input.PathPattern, input.SourceURL,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting robots directive for host %q in job %q: %w", input.Host, input.JobID, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("getting last insert ID for robots directive: %w", err)
	}

	return id, nil
}

// GetRobotsDirectivesByHost returns robots directives for a host within a job, up to limit rows.
func (db *DB) GetRobotsDirectivesByHost(jobID, host string, limit int) ([]RobotsDirective, error) {
	rows, err := db.Query(
		`SELECT `+robotsColumns+` FROM robots_directives
		 WHERE job_id = ? AND host = ?
		 ORDER BY id ASC LIMIT ?`,
		jobID, host, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying robots directives for host %q in job %q: %w", host, jobID, err)
	}
	defer rows.Close()

	directives := []RobotsDirective{}
	for rows.Next() {
		r, scanErr := scanRobotsDirective(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning robots directive row: %w", scanErr)
		}
		directives = append(directives, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating robots directive rows: %w", err)
	}

	return directives, nil
}
