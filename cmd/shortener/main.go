package main

import (
    "fmt"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/service"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/db"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
    server1, server2, log, databaseDSN, filePath := config.Settings()
    pool, err := db.Connect(databaseDSN)

    if err != nil {
        log.Fatal(fmt.Sprint(err))
    }

    defer pool.Close()

    if databaseDSN != "" {
        m, err := migrate.New("file://migrations", databaseDSN)

        if err != nil {
            log.Fatal(fmt.Sprintf("Ошибка загрузки миграций: %v", err))
        }

        if err := m.Up(); err != nil && err != migrate.ErrNoChange {
            log.Fatal(fmt.Sprintf("Ошибка запуска миграций: %v", err))
        }
    }

    store, err := storage.NewStorage(filePath, pool)

    if err != nil {
        log.Fatal(fmt.Sprint(err))
    }

    h := handler.NewHandler(store, server2, log)
    servers := []config.Server{server1, server2}

    service.NewService(h, servers, log).Run()
}
