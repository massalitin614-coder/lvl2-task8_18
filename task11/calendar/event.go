package calendar

import "time"

type Event struct {
	ID     int
	UserID int
	Date   time.Time
	Text   string
}
