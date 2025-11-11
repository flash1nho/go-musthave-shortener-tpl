package service

import (
		"context"
		"net/http"
		"os"
		"os/signal"
		"syscall"
		"time"
		"fmt"
		"compress/gzip"
		"slices"
		"sync"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
		"github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"

    "go.uber.org/zap"
)

type Service struct {
    handler *handler.Handler
    servers []config.Server
    log *zap.Logger
}

func NewService(handler *handler.Handler, servers []config.Server, log *zap.Logger) *Service {
    return &Service{
        handler: handler,
        servers: servers,
        log: log,
    }
}

func Decompressor(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Content-Encoding") == "gzip" {
            gzReader, err := gzip.NewReader(r.Body)
            if err != nil {
                http.Error(w, "Ошибка при распаковке gzip", http.StatusBadRequest)
                return
            }

            defer gzReader.Close()

            r.Body = gzReader
        }
        next.ServeHTTP(w, r)
    })
}

func (s *Service) mainRouter() http.Handler {
    r := chi.NewRouter()

    r.Use(middleware.Logger)
    r.Use(Decompressor)

    r.Post("/", s.handler.PostURLHandler)
    r.Post("/api/shorten", s.handler.APIShortenPostURLHandler)
    r.Get("/{id}", s.handler.GetURLHandler)
    r.Get("/ping", s.handler.Ping)
    r.Post("/api/shorten/batch", s.handler.APIShortenBatchPostURLHandler)

    return r
}

func runServer(s *Service, ctx context.Context, wg *sync.WaitGroup, addr string) {
	defer wg.Done()

	server := &http.Server{
		Addr: addr,
		Handler: s.mainRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		s.log.Info(fmt.Sprintf("Сервер запущен на http://%s", server.Addr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Fatal(fmt.Sprintf("Ошибка запуска сервера http://%s: %v", server.Addr, err))
		}
	}()

	select {
	case err := <-serverErr:
		s.log.Info(fmt.Sprint(err))
	case <-ctx.Done():
		s.log.Info(fmt.Sprintf("Завершение работы сервера http://%s", server.Addr))

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			s.log.Info(fmt.Sprintf("Ошибка завершения работы сервера http://%s: %v", server.Addr, err))
		} else {
			s.log.Info(fmt.Sprintf("Сервер http://%s успешно остановлен", server.Addr))
		}
	}
}

func (s *Service) Run() {
		var wg sync.WaitGroup
		ctx, cancel := context.WithCancel(context.Background())

    for _, server := range slices.Compact(s.servers) {
        wg.Add(1)
        go runServer(s, ctx, &wg, server.Addr)
    }

		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

		sig := <-signalChan
		s.log.Info(fmt.Sprintf("Полученный сигнал %s: инициирование завершения работы", sig))

		cancel()

		wg.Wait()

		s.log.Info("Все серверы успешно завершили работу.")
}
