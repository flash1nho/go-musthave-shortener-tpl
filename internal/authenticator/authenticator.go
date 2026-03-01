package authenticator

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
)

var hashKey = securecookie.GenerateRandomKey(32)
var secureCookieManager = securecookie.New(hashKey, nil)

const cookieName = "user_session_id"

type ctxUserID string

const CtxUserKey = ctxUserID("userID")

type AuthProvider interface {
	GetCookie(name string) (string, error)
	SetCookie(name, value string) error
}

func Authenticate(ctx context.Context, p AuthProvider) (context.Context, error) {
	var cookieValue string

	cookieValue, err := p.GetCookie(cookieName)

	var userID string

	if err != nil {
		userID, cookieValue, err = setSignedCookie()

		if err != nil {
			return nil, err
		}
	}

	p.SetCookie(cookieName, cookieValue)

	if userID == "" {
		userID, err = getUserIDFromCookie(cookieValue)

		if err != nil {
			return nil, err
		}
	}

	return context.WithValue(ctx, CtxUserKey, userID), nil
}

func setSignedCookie() (string, string, error) {
	userID, err := GenerateUniqueUserID()

	if err != nil {
		return "", "", err
	}

	encodedValue, err := secureCookieManager.Encode(cookieName, userID)

	if err != nil {
		return "", "", err
	}

	return userID, encodedValue, nil
}

func GenerateUniqueUserID() (string, error) {
	id, err := uuid.NewRandom()

	if err != nil {
		return "", fmt.Errorf("не удалось сгенерировать UUID: %w", err)
	}

	return id.String(), nil
}

func getUserIDFromCookie(cookieValue string) (string, error) {
	var userID string

	err := secureCookieManager.Decode(cookieName, cookieValue, &userID)

	if err != nil {
		return "", err
	}

	return userID, nil
}
