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

		"go-musthave-shortener-tpl/internal/handler"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

type Service struct {
    handler *handler.Handler
    hosts []string
}

func NewService(handler *handler.Handler, hosts []string) *Service {
    return &Service{
        handler: handler,
        hosts: hosts,
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

func startServer(host string, hostNum int, handler http.Handler, wg *sync.WaitGroup) *http.Server {
		defer wg.Done()

		srv := &http.Server{
			Addr: host,
			Handler: handler,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		log.Printf("Сервер %d запущен на http://%s", hostNum, host)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера %d: %v", hostNum, err)
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
	  hosts := slices.Compact(s.hosts)
		var servers []*http.Server
	  var wg sync.WaitGroup

		for hostNum, host := range hosts {
			hostNum++
			wg.Add(1)

			go func() {
				srv := startServer(host, hostNum, s.mainRouter(), &wg)
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
