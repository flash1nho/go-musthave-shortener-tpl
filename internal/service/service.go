package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-retryablehttp"

	"go.uber.org/zap"
)

type Service struct {
	handler     *handler.Handler
	servers     []config.Server
	log         *zap.Logger
	auditFile   string
	auditURL    string
	enableHTTPS *bool
}

func NewService(handler *handler.Handler, servers []config.Server, log *zap.Logger, auditFile string, auditURL string, enableHTTPS *bool) *Service {
	return &Service{
		handler:     handler,
		servers:     servers,
		log:         log,
		auditFile:   auditFile,
		auditURL:    auditURL,
		enableHTTPS: enableHTTPS,
	}
}

func (s *Service) mainRouter() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middlewares.Decompressor)
	r.Use(middlewares.Auth)

	r.Get("/ping", s.handler.Ping)
	r.Post("/api/shorten/batch", s.handler.APIShortenBatchPostURLHandler)
	r.Get("/api/user/urls", s.handler.APIUserURLHandler)
	r.Delete("/api/user/urls", s.handler.APIUserDeleteURLHandler)

	r.Group(func(r chi.Router) {
		subject := &middlewares.AuditSubject{}

		if s.auditFile != "" {
			subject.Register(&middlewares.FileObserver{FilePath: s.auditFile, Log: s.log})
		}

		if s.auditURL != "" {
			subject.Register(&middlewares.URLObserver{URL: s.auditURL, Log: s.log, Client: retryablehttp.NewClient()})
		}

		r.Use(middlewares.AuditMiddleware(subject))
		r.Post("/", s.handler.PostURLHandler)
		r.Post("/api/shorten", s.handler.APIShortenPostURLHandler)
		r.Get("/{id}", s.handler.GetURLHandler)
	})

	return r
}

func runServer(s *Service, ctx context.Context, wg *sync.WaitGroup, addr string) {
	defer wg.Done()

	server := &http.Server{
		Addr:         addr,
		Handler:      s.mainRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		if *s.enableHTTPS {
			s.log.Info(fmt.Sprintf("Сервер запущен на https://%s", server.Addr))
			err := server.ListenAndServeTLS("cert.pem", "key.pem")

			if err != nil && err != http.ErrServerClosed {
				s.log.Error(fmt.Sprintf("Ошибка запуска сервера https://%s: %v", server.Addr, err))
			}
		} else {
			s.log.Info(fmt.Sprintf("Сервер запущен на http://%s", server.Addr))
			err := server.ListenAndServe()

			if err != nil && err != http.ErrServerClosed {
				s.log.Error(fmt.Sprintf("Ошибка запуска сервера http://%s: %v", server.Addr, err))
			}
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
			s.log.Error(fmt.Sprintf("Ошибка завершения работы сервера http://%s: %v", server.Addr, err))
		} else {
			s.log.Info(fmt.Sprintf("Сервер http://%s успешно остановлен", server.Addr))
		}

		s.log.Info("Сохранение данных в хранилище...")

		if err := s.handler.Store.Close(); err != nil {
			s.log.Error(fmt.Sprintf("Ошибка при сохранении данных: %v", err))
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
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	sig := <-signalChan
	s.log.Info(fmt.Sprintf("Полученный сигнал %s: инициирование завершения работы", sig))

	cancel()

	wg.Wait()

	s.log.Info("Все серверы успешно завершили работу.")
}
