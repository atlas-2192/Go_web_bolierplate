package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Session struct {
	dbpool *pgxpool.Pool
}

// NewSession returns a new Session.
func NewSession(dbpool *pgxpool.Pool) *Session {
	return &Session{dbpool: dbpool}
}

// GetCurrentTime returns the current time from the database.
func GetCurrentTime(ctx context.Context, session *Session) (time.Time, error) {
	var currentTime time.Time
	err := session.dbpool.QueryRow(ctx, "select now()").Scan(&currentTime)
	if err != nil {
		return time.Time{}, err
	}

	return currentTime, nil
}
