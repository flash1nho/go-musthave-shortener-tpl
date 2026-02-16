package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/service"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	server1, server2, log, databaseDSN, filePath, auditFile, auditURL, enableHTTPS := config.Settings()
	store, err := storage.NewStorage(filePath, databaseDSN)

	if err != nil {
		log.Error(fmt.Sprint(err))
	}

	h := handler.NewHandler(store, server2, log)
	servers := []config.Server{server1, server2}

	service.NewService(h, servers, log, auditFile, auditURL, enableHTTPS).Run()
}
