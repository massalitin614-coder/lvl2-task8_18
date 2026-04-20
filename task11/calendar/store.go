package calendar

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("event not fount")
)

type Store struct {
	mu     sync.RWMutex
	events map[int]Event
	nextID int
}

func NewStore() *Store {
	return &Store{
		events: make(map[int]Event),
		nextID: 1,
	}
}

// создаем
func (s *Store) Create(userID int, date time.Time, text string) Event {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := Event{
		ID:     s.nextID,
		UserID: userID,
		Date:   date,
		Text:   text,
	}

	s.events[s.nextID] = e
	s.nextID++
	return e
}

// обновляем
func (s *Store) Update(id int, text string) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.events[id]
	if !ok {
		return Event{}, ErrNotFound
	}
	e.Text = text
	s.events[id] = e
	return e, nil
}

// удаление
func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[id]; !ok {
		return ErrNotFound
	}
	delete(s.events, id)

	return nil
}

func (s *Store) EventsForDay(userID int, date time.Time) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Event

	for _, e := range s.events {
		if e.UserID == userID && sameDay(e.Date, date) {
			result = append(result, e)
		}
	}

	return result
}

func (s *Store) EventsForWeek(userID int, start time.Time) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Event

	end := start.AddDate(0, 0, 7)

	for _, e := range s.events {
		if e.UserID != userID {
			continue
		}

		if !e.Date.Before(start) && e.Date.Before(end) {
			result = append(result, e)
		}
	}

	return result
}

func (s *Store) EventsForMonth(userID int, start time.Time) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Event

	end := start.AddDate(0, 1, 0)

	for _, e := range s.events {
		if e.UserID == userID &&
			!e.Date.Before(start) &&
			e.Date.Before(end) {
			result = append(result, e)
		}
	}

	return result
}
func sameDay(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
