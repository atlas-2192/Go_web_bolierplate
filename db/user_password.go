package db

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgxutil"
	"golang.org/x/crypto/argon2"
)

const (
	passwordSaltLen          = 16
	argon2DefaultIterations  = 1
	argon2DefaultMemoryKiB   = 64 * 1024
	argon2DefaultParallelism = 1
	argon2DefaultKeyLen      = 32
)

// SetUserPassword sets the password for a user.
func SetUserPassword(ctx context.Context, db pgxutil.DB, userID uuid.UUID, password string) error {
	randBytes := make([]byte, passwordSaltLen)
	_, err := rand.Read(randBytes)
	if err != nil {
		return err
	}

	digest := argon2.IDKey([]byte(password), randBytes, argon2DefaultIterations, argon2DefaultMemoryKiB, argon2DefaultParallelism, argon2DefaultKeyLen)

	_, err = db.Exec(
		ctx,
		`insert into user_passwords (user_id, algorithm, salt, min_memory, iterations, parallelism, digest)
values ($1, $2, $3, $4, $5, $6, $7)
on conflict (user_id) do update
set algorithm = excluded.algorithm,
	salt = excluded.salt,
	min_memory = excluded.min_memory,
	iterations = excluded.iterations,
	parallelism = excluded.parallelism,
	digest = excluded.digest`,
		userID, "Argon2id", randBytes, argon2DefaultMemoryKiB, argon2DefaultIterations, argon2DefaultParallelism, digest)
	if err != nil {
		return err
	}

	return nil
}

var ErrPasswordIncorrect = errors.New("password is incorrect")
var ErrPasswordMissing = errors.New("password is missing")

func ValidateUserPassword(ctx context.Context, db pgxutil.DB, userID uuid.UUID, password string) error {
	var algorithm string
	var salt []byte
	var minMemory uint32
	var iterations uint32
	var parallelism uint8
	var digest []byte

	err := db.QueryRow(
		ctx,
		`select algorithm, salt, min_memory, iterations, parallelism, digest
from user_passwords
where user_id = $1`,
		userID).Scan(&algorithm, &salt, &minMemory, &iterations, &parallelism, &digest)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPasswordMissing
		}
		return err
	}

	passwordDigest := argon2.IDKey([]byte(password), salt, iterations, minMemory, parallelism, argon2DefaultKeyLen)

	if subtle.ConstantTimeCompare(digest, passwordDigest) != 1 {
		return ErrPasswordIncorrect
	}

	return nil
}
