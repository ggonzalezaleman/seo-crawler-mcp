package storage

import "testing"

func TestInsertAndGetRobotsDirectives(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	id, err := db.InsertRobotsDirective(RobotsDirectiveInput{
		JobID:       job.ID,
		Host:        "example.com",
		UserAgent:   "*",
		RuleType:    "disallow",
		PathPattern: "/admin",
		SourceURL:   "https://example.com/robots.txt",
	})
	if err != nil {
		t.Fatalf("InsertRobotsDirective: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero directive ID")
	}

	_, err = db.InsertRobotsDirective(RobotsDirectiveInput{
		JobID:       job.ID,
		Host:        "example.com",
		UserAgent:   "Googlebot",
		RuleType:    "allow",
		PathPattern: "/",
		SourceURL:   "https://example.com/robots.txt",
	})
	if err != nil {
		t.Fatalf("InsertRobotsDirective 2: %v", err)
	}

	// Different host — should not appear in results.
	_, err = db.InsertRobotsDirective(RobotsDirectiveInput{
		JobID:       job.ID,
		Host:        "other.com",
		UserAgent:   "*",
		RuleType:    "disallow",
		PathPattern: "/",
		SourceURL:   "https://other.com/robots.txt",
	})
	if err != nil {
		t.Fatalf("InsertRobotsDirective 3: %v", err)
	}

	directives, err := db.GetRobotsDirectivesByHost(job.ID, "example.com", 1000)
	if err != nil {
		t.Fatalf("GetRobotsDirectivesByHost: %v", err)
	}
	if len(directives) != 2 {
		t.Fatalf("expected 2 directives for example.com, got %d", len(directives))
	}
	if directives[0].PathPattern != "/admin" {
		t.Errorf("expected path_pattern %q, got %q", "/admin", directives[0].PathPattern)
	}
	if directives[1].UserAgent != "Googlebot" {
		t.Errorf("expected user_agent %q, got %q", "Googlebot", directives[1].UserAgent)
	}
}
