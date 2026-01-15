package middlewares

import (
	  "compress/gzip"
	  "net/http"
	  "context"
	  "fmt"
	  "time"
	  "os"
	  "encoding/json"
	  "bytes"

    "github.com/gorilla/securecookie"
    "github.com/google/uuid"
)

var hashKey = securecookie.GenerateRandomKey(32)
var secureCookieManager = securecookie.New(hashKey, nil)

const cookieName = "user_session_id"

type ctxUserID string
const CtxUserKey = ctxUserID("userID")

type AuditEvent struct {
		Timestamp int64     `json:"ts"`
		Action    string    `json:"action"`
		UserID    string    `json:"user_id"`
		URL       string    `json:"url"`
}

type Observer interface {
		Notify(event AuditEvent)
}

type AuditSubject struct {
		observers []Observer
}

type FileObserver struct {
		FilePath string
}

type URLObserver struct {
		URL string
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

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)

    var userID string

		if err != nil {
				userID, err = setSignedCookie(w)

				if err != nil {
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
				}
		}

		if userID == "" {
				userID, err = getUserIDFromCookie(w, cookie.Value)

				if err != nil {
						http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
						return
				}
		}

		ctx := context.WithValue(r.Context(), CtxUserKey, userID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserIDFromCookie(w http.ResponseWriter, cookieValue string) (string, error) {
		var userID string

		err := secureCookieManager.Decode(cookieName, cookieValue, &userID)

		if err != nil {
				return "", err
		}

		return userID, nil
}

func setSignedCookie(w http.ResponseWriter) (string, error) {
	  userID, err := GenerateUniqueUserID()

	  if err != nil {
	  		return "", err
	  }

		encodedValue, err := secureCookieManager.Encode(cookieName, userID)

		if err != nil {
				return "", err
		}

		cookie := &http.Cookie{
				Name:     cookieName,
				Value:    encodedValue,
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   3600 * 24 * 7,
		}

		http.SetCookie(w, cookie)

		return userID, nil
}

func GenerateUniqueUserID() (string, error) {
		id, err := uuid.NewRandom()

		if err != nil {
				return "", fmt.Errorf("не удалось сгенерировать UUID: %w", err)
		}

		return id.String(), nil
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
				return
		}

		defer file.Close()

		data, _ := json.Marshal(e)
		file.Write(append(data, '\n'))
}

func (u *URLObserver) Notify(e AuditEvent) {
		data, _ := json.Marshal(e)
		resp, err := http.Post(u.URL, "application/json", bytes.NewBuffer(data))

		if err != nil {
				return
		}

		resp.Body.Close()
}

func AuditMiddleware(subject *AuditSubject) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						userID, _ := r.Context().Value(CtxUserKey).(string)

						next.ServeHTTP(w, r)

						event := AuditEvent{
								Timestamp: time.Now().Unix(),
								Action:    r.Method,
								UserID:    userID,
								URL:    	 r.URL.Path,
						}

						subject.NotifyAll(event)
				})
		}
}
