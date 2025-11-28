package helpers

import (
		"crypto/sha256"
    "encoding/base64"
    "fmt"
    "net/http"

    "github.com/gorilla/securecookie"
    "github.com/google/uuid"
)

var hashKey = securecookie.GenerateRandomKey(32)
var secureCookieManager = securecookie.New(hashKey, nil)

const cookieName = "user_session_id"

func GetUserIDFromCookie(r *http.Request) (string, error) {
		if cookie, err := r.Cookie(cookieName); err == nil {
			var userID string

			if err = secureCookieManager.Decode(cookieName, cookie.Value, &userID); err == nil {
				return userID, nil
			}

			return "", fmt.Errorf("недействительная или подделанная кука: %w", err)
		}

		return "", http.ErrNoCookie
}

func SetSignedCookie(w http.ResponseWriter, userID string) error {
		encodedValue, err := secureCookieManager.Encode(cookieName, userID)

		if err != nil {
			return err
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

		return nil
}

func GenerateUniqueUserID() (string, error) {
		id, err := uuid.NewRandom()

		if err != nil {
			return "", fmt.Errorf("не удалось сгенерировать UUID: %w", err)
		}

		return id.String(), nil
}

func GenerateShortURL(value string) string {
    h := sha256.New()
    h.Write([]byte(value))
    bs := h.Sum(nil)
    hash := base64.URLEncoding.EncodeToString(bs[:])

    return hash[:8]
}
