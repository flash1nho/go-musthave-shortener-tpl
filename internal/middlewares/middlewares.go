package middlewares

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/authenticator"

	"go.uber.org/zap"
)

type AuditEvent struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
}

type httpProvider struct {
	w http.ResponseWriter
	r *http.Request
}

type Observer interface {
	Notify(event AuditEvent)
}

type AuditSubject struct {
	observers []Observer
}

type FileObserver struct {
	mu       sync.RWMutex
	FilePath string
	Log      *zap.Logger
}

type URLObserver struct {
	URL    string
	Log    *zap.Logger
	Client *retryablehttp.Client
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

func (s *AuditSubject) Register(o Observer) {
	s.observers = append(s.observers, o)
}

func (s *AuditSubject) NotifyAll(e AuditEvent) {
	for _, o := range s.observers {
		go o.Notify(e) // Запускаем в горутинах, чтобы не блокировать ответ пользователю
	}
}

func (f *FileObserver) Notify(e AuditEvent) {
	file, err := os.OpenFile(f.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		f.Log.Error(fmt.Sprint(err))
		return
	}

	defer file.Close()

	data, err := json.Marshal(e)

	if err != nil {
		f.Log.Error(fmt.Sprint(err))
		return
	}

	f.mu.Lock()

	file.Write(append(data, '\n'))

	f.mu.Unlock()
}

func (u *URLObserver) Notify(e AuditEvent) {
	data, err := json.Marshal(e)

	if err != nil {
		u.Log.Error(fmt.Sprint(err))
		return
	}

	retryClient := u.Client
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1
	retryClient.RetryWaitMax = 5

	req, err := retryablehttp.NewRequest("POST", u.URL, bytes.NewBuffer(data))

	if err != nil {
		u.Log.Error(fmt.Sprint(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := retryClient.Do(req)

	if err != nil {
		u.Log.Error(fmt.Sprint(err))
		return
	}

	resp.Body.Close()
}

func Audit(subject *AuditSubject) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, _ := r.Context().Value(authenticator.CtxUserKey).(string)

			next.ServeHTTP(w, r)

			event := AuditEvent{
				Timestamp: time.Now().Unix(),
				Action:    r.Method,
				UserID:    userID,
				URL:       r.URL.Path,
			}

			subject.NotifyAll(event)
		})
	}
}

func TrustedSubnet(trustedSubnet string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trustedSubnet == "" {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			_, subnet, err := net.ParseCIDR(trustedSubnet)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			ipStr := r.Header.Get("X-Real-IP")
			ip := net.ParseIP(ipStr)

			if ip == nil || !subnet.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (p *httpProvider) GetCookie(cookieName string) (string, error) {
	cookie, err := p.r.Cookie(cookieName)

	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func (p *httpProvider) SetCookie(cookieName, cookieValue string) error {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600 * 24 * 7,
	}

	http.SetCookie(p.w, cookie)

	return nil
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, err := authenticator.Authenticate(r.Context(), &httpProvider{w, r})

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.Clone(ctx))
	})
}
