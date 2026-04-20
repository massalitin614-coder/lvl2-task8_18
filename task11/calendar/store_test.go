package calendar

import (
	"testing"
	"time"
)

func TestCreateAndGetByDay(t *testing.T) {
	store := NewStore()

	day := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)

	store.Create(1, day.Add(2*time.Hour), "event1")
	store.Create(1, day.Add(5*time.Hour), "event2")
	store.Create(1, day.AddDate(0, 0, 1), "event3")

	events := store.EventsForDay(1, day)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestEventsForWeek(t *testing.T) {
	store := NewStore()

	start := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)

	store.Create(1, start.AddDate(0, 0, 1), "event1")
	store.Create(1, start.AddDate(0, 0, 3), "event2")
	store.Create(1, start.AddDate(0, 0, 10), "event3")

	events := store.EventsForWeek(1, start)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestDelete(t *testing.T) {
	store := NewStore()

	e := store.Create(1, time.Now(), "test")

	if err := store.Delete(e.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Delete(e.ID); err == nil {
		t.Fatalf("expected error on second delete")
	}
}
