package middlewares

import (
	  "compress/gzip"
	  "net/http"
	  "context"
	  "fmt"

    "github.com/gorilla/securecookie"
    "github.com/google/uuid"
)

var hashKey = securecookie.GenerateRandomKey(32)
var secureCookieManager = securecookie.New(hashKey, nil)

const cookieName = "user_session_id"

type ctxUserID string
const CtxUserKey = ctxUserID("userID")

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
