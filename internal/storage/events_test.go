package storage

import "testing"

func TestInsertAndGetEvents(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	details := `{"pages":10}`
	url := "https://example.com/"
	id, err := db.InsertEvent(job.ID, "crawl_started", &details, &url)
	if err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero event ID")
	}

	_, err = db.InsertEvent(job.ID, "page_fetched", nil, &url)
	if err != nil {
		t.Fatalf("InsertEvent 2: %v", err)
	}

	events, err := db.GetEventsByJob(job.ID, 10)
	if err != nil {
		t.Fatalf("GetEventsByJob: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Most recent first (DESC order).
	if events[0].EventType != "page_fetched" {
		t.Errorf("expected first event %q, got %q", "page_fetched", events[0].EventType)
	}
}

func TestGetEventsByJobLimit(t *testing.T) {
	db := testDB(t)

	job, err := db.CreateJob("crawl", "{}", "[]")
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	for i := range 5 {
		_, err := db.InsertEvent(job.ID, "tick", nil, nil)
		if err != nil {
			t.Fatalf("InsertEvent %d: %v", i, err)
		}
	}

	events, err := db.GetEventsByJob(job.ID, 3)
	if err != nil {
		t.Fatalf("GetEventsByJob: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events with limit, got %d", len(events))
	}
}
