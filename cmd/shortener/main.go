package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/facade"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/grpc"
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

	settings := config.Settings()
	store, err := storage.NewStorage(settings.FilePath, settings.DatabaseDSN)

	if err != nil {
		settings.Log.Error(fmt.Sprint(err))
	}

	f := facade.NewFacade(store, settings.Server2.BaseURL)
	h := handler.NewHandler(f, settings)
	gh := grpc.NewHandler(f)
	service.NewService(h, gh, settings).Run()
}
