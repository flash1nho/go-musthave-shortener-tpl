package main

import (
    "fmt"
    "net/http"
    _ "net/http/pprof"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/service"
)

func main() {
    go func() {
        http.ListenAndServe("localhost:6060", nil)
    }()

    server1, server2, log, databaseDSN, filePath, auditFile, auditURL := config.Settings()
    store, err := storage.NewStorage(filePath, databaseDSN)

    if err != nil {
        log.Error(fmt.Sprint(err))
    }

    h := handler.NewHandler(store, server2, log)
    servers := []config.Server{server1, server2}

    service.NewService(h, servers, log, auditFile, auditURL).Run()
}
