package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	pb "github.com/flash1nho/go-musthave-shortener-tpl/internal/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/handler"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-retryablehttp"

	"go.uber.org/zap"
)

type Service struct {
	handler       *handler.Handler
	gHandler      *pb.GrpcHandler
	servers       []config.Server
	log           *zap.Logger
	auditFile     string
	auditURL      string
	enableHTTPS   bool
	trustedSubnet string
}

func NewService(handler *handler.Handler, gHandler *pb.GrpcHandler, settings config.SettingsObject) *Service {
	servers := []config.Server{settings.Server1, settings.Server2}

	return &Service{
		handler:       handler,
		gHandler:      gHandler,
		servers:       servers,
		log:           settings.Log,
		auditFile:     settings.AuditFile,
		auditURL:      settings.AuditURL,
		enableHTTPS:   settings.EnableHTTPS,
		trustedSubnet: settings.TrustedSubnet,
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

		r.Use(middlewares.Audit(subject))
		r.Post("/", s.handler.PostURLHandler)
		r.Post("/api/shorten", s.handler.APIShortenPostURLHandler)
		r.Get("/{id}", s.handler.GetURLHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(middlewares.TrustedSubnet(s.trustedSubnet))
		r.Get("/api/internal/stats", s.handler.APIInternalStats)
	})

	return r
}

func runServer(ctx context.Context, s *Service, addr string) {
	server := &http.Server{
		Addr:         addr,
		Handler:      s.mainRouter(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErr := make(chan error, 1)

	go func() {
		if s.enableHTTPS {
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
	}
}

func runGrpcServer(ctx context.Context, s *Service) {
	serverErr := make(chan error, 1)
	creds := insecure.NewCredentials()
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(pb.Auth),
	)

	go func() {
		listen, err := net.Listen("tcp", ":3200")

		if err == nil {
			pb.RegisterShortenerServiceServer(grpcServer, s.gHandler)

			s.log.Info("сервер gRPC начал работу")

			if err := grpcServer.Serve(listen); err != nil {
				s.log.Error(fmt.Sprintf("Ошибка при работе gRPC сервера: %v", err))
			}
		} else {
			s.log.Error(fmt.Sprintf("Ошибка при инициализации gRPC listener: %v", err))
		}
	}()

	select {
	case err := <-serverErr:
		s.log.Info(fmt.Sprint(err))
	case <-ctx.Done():
		s.log.Info("Завершение работы gRPC сервера")

		grpcServer.GracefulStop()

		s.log.Info("gRPC сервер успешно остановлен")
	}
}

func (s *Service) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)

	for _, server := range slices.Compact(s.servers) {
		srv := server

		g.Go(func() error {
			runServer(ctx, s, srv.Addr)
			return nil
		})
	}

	g.Go(func() error {
		runGrpcServer(ctx, s)
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		s.log.Error(fmt.Sprintf("Работа завершена с ошибкой: %v", err))
	}

	s.log.Info("Сохранение данных в хранилище...")

	if err := s.handler.Facade.Store.Close(); err != nil {
		s.log.Error(fmt.Sprintf("Ошибка при сохранении данных: %v", err))
	}

	s.log.Info("Все серверы успешно завершили работу.")
}
