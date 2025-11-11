package main

import (
	  "fmt"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/service"
)

func main() {
	server1, server2, log, databaseDSN, filePath := config.Settings()
    store, err := storage.NewStorage(databaseDSN, filePath)

    if err != nil {
        log.Fatal(fmt.Sprint(err))
    }

    h := handler.NewHandler(store, server2)
    servers := []config.Server{server1, server2}

    service.NewService(h, servers, log).Run()
}
