package db

import (
		"context"
		"fmt"

		"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseDSN string) (*pgxpool.Pool, error) {
		pool, err := pgxpool.New(context.Background(), databaseDSN)

		if err != nil {
			return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
		}

		return pool, nil
}
