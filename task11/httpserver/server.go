package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"task11/calendar"
	"time"
)

type Server struct {
	store *calendar.Store
}

func NewServer(store *calendar.Store) *Server {
	return &Server{store: store}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/create_event", s.createEvent)
	mux.HandleFunc("/update_event", s.updateEvent)
	mux.HandleFunc("/delete_event", s.deleteEvent)
	mux.HandleFunc("/events_for_day", s.eventsForDay)
	mux.HandleFunc("/events_for_week", s.eventsForWeek)
	mux.HandleFunc("/events_for_month", s.eventsForMonth)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "applacation/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{
		"error": msg,
	})
}

func parseCreateRequest(r *http.Request) (int, time.Time, string, error) {
	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil {
		return 0, time.Time{}, "", err
	}

	date, err := time.Parse("2006-01-02", r.FormValue("date"))
	if err != nil {
		return 0, time.Time{}, "", err
	}

	text := r.FormValue("event")

	return userID, date, text, nil
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		writeError(w, 400, "invalid method")
		return
	}

	userID, data, text, err := parseCreateRequest(r)
	if err != nil {
		writeError(w, 400, "invalid input")
		return
	}
	event := s.store.Create(userID, data, text)
	writeJSON(w, 200, map[string]any{
		"result": event,
	})
}

func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 400, "invalid method")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, 400, "invalid input")
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		writeError(w, 400, "invlid id")
		return
	}

	text := r.FormValue("event")

	event, err := s.store.Update(id, text)
	if err != nil {
		writeError(w, 503, err.Error())
		return
	}
	writeJSON(w, 200, map[string]any{
		"result": event,
	})
}

func (s *Server) deleteEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 400, "invalid method")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, 400, "invalid input")
		return
	}

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	err = s.store.Delete(id)
	if err != nil {
		writeError(w, 503, err.Error())
		return
	}

	writeJSON(w, 200, map[string]string{
		"result": "deleted",
	})
}

func (s Server) eventsForDay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 400, "invalid method")
		return
	}

	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, 400, "invalid user_id")
		return
	}

	date, err := time.Parse("2006-01-02", r.URL.Query().Get("date"))
	if err != nil {
		writeError(w, 400, "invalid date")
		return
	}

	events := s.store.EventsForDay(userID, date)

	writeJSON(w, 200, map[string]any{
		"result": events,
	})
}

func (s *Server) eventsForWeek(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 400, "invalid method")
		return
	}

	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, 400, "invalid user_id")
		return
	}

	dateStr := r.URL.Query().Get("date")

	start, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, 400, "invalid date")
		return
	}

	events := s.store.EventsForWeek(userID, start)

	writeJSON(w, 200, map[string]any{
		"result": events,
	})
}

func (s *Server) eventsForMonth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 400, "invalid method")
		return
	}

	userID, err := strconv.Atoi(r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, 400, "invalid user_id")
		return
	}

	dateStr := r.URL.Query().Get("date")

	start, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, 400, "invalid date")
		return
	}

	events := s.store.EventsForMonth(userID, start)

	writeJSON(w, 200, map[string]any{
		"result": events,
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf("%s %s %v\n", r.Method, r.URL.Path, time.Since(start))
	})
}
