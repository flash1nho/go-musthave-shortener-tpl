package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(databaseDSN string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseDSN)

	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %w", err)
	}

	err = pool.Ping(ctx)

	if err != nil {
		return nil, fmt.Errorf("ошибка пинга базы данных: %w", err)
	}

	return pool, nil
}
