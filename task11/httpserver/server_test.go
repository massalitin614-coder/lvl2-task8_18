package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"task11/calendar"
	"testing"
)

func TestCreateEventHandler(t *testing.T) {
	store := calendar.NewStore()
	server := NewServer(store)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	body := "user_id=1&date=2026-04-19&event=test"

	req := httptest.NewRequest(http.MethodPost, "/create_event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
