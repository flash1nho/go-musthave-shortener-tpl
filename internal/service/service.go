package service

import (
		"context"
		"log"
		"net/http"
		"os"
		"os/signal"
		"sync"
		"syscall"
		"time"
		"slices"

    "go-musthave-shortener-tpl/internal/config"
		"go-musthave-shortener-tpl/internal/handler"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

type Service struct {
    handler *handler.Handler
    servers []config.Server
}

func NewService(handler *handler.Handler, servers []config.Server) *Service {
    return &Service{
        handler: handler,
        servers: servers,
    }
}

func (s *Service) mainRouter() http.Handler {
		r := chi.NewRouter()
	  r.Use(middleware.Logger)
	  r.Use(middleware.Recoverer)
	  r.Post("/", s.handler.PostURLHandler)
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

		log.Printf("Сервер %d запущен на %s", serverNum, BaseURL)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера %d: %v", serverNum, err)
		}

		return srv
}

func stopServer(s *http.Server, shutdownWg *sync.WaitGroup) {
		defer shutdownWg.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			log.Printf("Ошибка при плавном отключении сервера %s: %v", s.Addr, err)
		}

		log.Printf("Сервер %s успешно завершил работу.", s.Addr)
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

    log.Printf("Получен сигнал завершения. Начинаю плавное отключение...")

		var shutdownWg sync.WaitGroup

		for _, srv := range servers {
			shutdownWg.Add(1)

			go func() {
				stopServer(srv, &shutdownWg)
			}()
		}

		shutdownWg.Wait()

		log.Printf("Все серверы успешно завершили работу.")
}
