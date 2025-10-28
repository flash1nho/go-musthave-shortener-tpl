package main

import (
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/service"
)

func main() {
    store := storage.NewStorage()
    server1, server2 := config.Servers()
    h := handler.NewHandler(store, server2)
    servers := []config.Server{server1, server2}
    service.NewService(h, servers).Run()
}
