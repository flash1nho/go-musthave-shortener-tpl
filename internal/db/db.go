package db

import (
		"context"

		"github.com/jackc/pgx/v5"
)

func Connect(databaseDSN string) (*pgx.Conn, error) {
		conn, err := pgx.Connect(context.Background(), databaseDSN)
		if err != nil {
			return nil, err
		}
		defer conn.Close(context.Background())

		return conn, nil
}
