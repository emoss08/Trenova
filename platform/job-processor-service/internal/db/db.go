package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func InitDB(ctx context.Context, connectionString string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, connectionString)

	return conn, err
}
