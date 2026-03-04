package authenticator

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
)

const cookieName = "user_session_id"

type UserID string

const userKey = UserID("userID")

type Authenticator struct {
	cookieManager *securecookie.SecureCookie
}

type CookieData struct {
	userID      string
	cookieValue string
}

func NewAuthenticator() *Authenticator {
	return &Authenticator{
		cookieManager: securecookie.New(securecookie.GenerateRandomKey(32), nil),
	}
}

type AuthProvider interface {
	GetCookie(ctx context.Context, name string) (string, error)
	SetCookie(ctx context.Context, name string, value string) error
}

func FromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userKey).(string)

	if !ok {
		return "", fmt.Errorf("userID не найден в контексте")
	}

	return userID, nil
}

func GetUserKey() UserID {
	return userKey
}

func (a *Authenticator) Authenticate(ctx context.Context, p AuthProvider) (context.Context, error) {
	var cookieValue string

	cookieValue, err := p.GetCookie(ctx, cookieName)

	var userID string

	if err != nil {
		cookieData, err := a.createSignedCookie()

		if err != nil {
			return nil, err
		}

		userID = cookieData.userID
		cookieValue = cookieData.cookieValue
	}

	if userID == "" {
		userID, err = a.getUserIDFromCookie(cookieValue)

		if err != nil {
			return nil, err
		}
	}

	p.SetCookie(ctx, cookieName, cookieValue)

	return context.WithValue(ctx, userKey, userID), nil
}

func (a *Authenticator) createSignedCookie() (*CookieData, error) {
	userID, err := GenerateUniqueUserID()

	if err != nil {
		return nil, err
	}

	cookieValue, err := a.cookieManager.Encode(cookieName, userID)

	if err != nil {
		return nil, fmt.Errorf("ошибка кодирования cookie: %w", err)
	}

	return &CookieData{userID: userID, cookieValue: cookieValue}, nil
}

func (a *Authenticator) getUserIDFromCookie(cookieValue string) (string, error) {
	var userID string

	err := a.cookieManager.Decode(cookieName, cookieValue, &userID)

	if err != nil {
		return "", fmt.Errorf("ошибка декодирования cookie: %w", err)
	}

	return userID, nil
}

func GenerateUniqueUserID() (string, error) {
	id, err := uuid.NewRandom()

	if err != nil {
		return "", fmt.Errorf("не удалось сгенерировать UUID: %w", err)
	}

	return id.String(), nil
}
