package main

import (
    "go-musthave-shortener-tpl/internal/storage"
    "go-musthave-shortener-tpl/internal/config"
    "go-musthave-shortener-tpl/internal/handler"
    "go-musthave-shortener-tpl/internal/service"
)

func main() {
    store := storage.NewStorage()
    server1, server2 := config.Servers()
    h := handler.NewHandler(store, server2)
    servers := []config.Server{server1, server2}
    service.NewService(h, servers).Run()
}
