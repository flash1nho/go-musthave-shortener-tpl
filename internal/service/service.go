package service

import (
		"context"
		"net/http"
		"os"
		"os/signal"
		"sync"
		"syscall"
		"time"
		"slices"
		"fmt"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
		"github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
		"github.com/flash1nho/go-musthave-shortener-tpl/internal/logger"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

type Service struct {
    handler *handler.Handler
    servers []config.Server
}

func NewService(handler *handler.Handler, servers []config.Server) *Service {
	  logger.Initialize("info")

    return &Service{
        handler: handler,
        servers: servers,
    }
}

func (s *Service) mainRouter() http.Handler {
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Post("/", s.handler.PostURLHandler)
    r.Post("/api/shorten", s.handler.ApiShortenPostURLHandler)
    r.Get("/{id}", s.handler.GetURLHandler)

    return r
}

func startServer(addr string, BaseURL string, serverNum int, handler http.Handler, wg *sync.WaitGroup) *http.Server {
		defer wg.Done()

		srv := &http.Server{
			Addr: addr,
			Handler: handler,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		logger.Log.Info(fmt.Sprintf("Сервер %d запущен на http://%s", serverNum, addr))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Log.Fatal(fmt.Sprintf("Ошибка запуска сервера %s: %v", serverNum, err))
		}

		return srv
}

func stopServer(srv *http.Server, shutdownWg *sync.WaitGroup) {
		defer shutdownWg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
        logger.Log.Error(fmt.Sprintf("Ошибка при плавном отключении сервера %s: %v", srv.Addr, err))
		}

		logger.Log.Info(fmt.Sprintf("Сервер %s успешно завершил работу.", srv.Addr))
}

func (s *Service) Run() {
	  configServers := slices.Compact(s.servers)
		var servers []*http.Server
	  var wg sync.WaitGroup

		for serverNum, server := range configServers {
			serverNum++
			wg.Add(1)

			go func() {
				srv := startServer(server.Addr, server.BaseURL, serverNum, s.mainRouter(), &wg)
				servers = append(servers, srv)
			}()
		}

		stop := make(chan os.Signal, 1)
	  signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	  <-stop

    logger.Log.Info("Получен сигнал завершения. Начинаю плавное отключение...")

		var shutdownWg sync.WaitGroup

		for _, srv := range servers {
			shutdownWg.Add(1)

			go func() {
				stopServer(srv, &shutdownWg)
			}()
		}

		shutdownWg.Wait()

		logger.Log.Info("Все серверы успешно завершили работу.")
}
