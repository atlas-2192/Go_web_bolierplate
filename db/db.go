package db

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxutil"
)

type Session struct {
	dbpool *pgxpool.Pool
}

// NewSession returns a new Session.
func NewSession(dbpool *pgxpool.Pool) *Session {
	return &Session{dbpool: dbpool}
}

func DBPool(session *Session) *pgxpool.Pool {
	return session.dbpool
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

type User struct {
	ID       uuid.UUID
	Username string
}

func GetUserByUsername(ctx context.Context, session *Session, username string) (*User, error) {
	user, err := pgxutil.SelectRow(ctx, session.dbpool, "select id, username from users where username = $1", []any{username}, pgx.RowToAddrOfStructByPos[User])
	if err != nil {
		return nil, err
	}

	return user, nil
}
