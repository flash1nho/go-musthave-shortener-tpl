package main

import (
    "go-musthave-shortener-tpl/internal/storage"
    "go-musthave-shortener-tpl/internal/config"
    "go-musthave-shortener-tpl/internal/handler"
    "go-musthave-shortener-tpl/internal/service"
)

func main() {
    store := storage.NewStorage()
    host1, host2 := config.Hosts()
    h := handler.NewHandler(store, host2)
    hosts := []string{host1, host2}
    service.NewService(h, hosts).Run()
}
