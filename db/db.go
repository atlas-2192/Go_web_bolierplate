package db

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxutil"
)

// GetCurrentTime returns the current time from the database.
func GetCurrentTime(ctx context.Context, db pgxutil.DB) (time.Time, error) {
	var currentTime time.Time
	err := db.QueryRow(ctx, "select now()").Scan(&currentTime)
	if err != nil {
		return time.Time{}, err
	}

	return currentTime, nil
}

type User struct {
	ID       uuid.UUID
	Username string
}

func GetUserByUsername(ctx context.Context, db pgxutil.DB, username string) (*User, error) {
	user, err := pgxutil.SelectRow(ctx, db, "select id, username from users where username = $1", []any{username}, pgx.RowToAddrOfStructByPos[User])
	if err != nil {
		return nil, err
	}

	return user, nil
}
