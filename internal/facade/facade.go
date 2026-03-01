package facade

import (
	"context"
	"fmt"
	"net/url"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/authenticator"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/helpers"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
)

type Facade struct {
	Store   *storage.Storage
	BaseURL string
}

type BatchUserShortenResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func NewFacade(store *storage.Storage, BaseURL string) *Facade {
	return &Facade{
		Store:   store,
		BaseURL: BaseURL,
	}
}

func (f *Facade) PostURLFacade(userID string, originalURL string) (string, error) {
	shortURL := helpers.GenerateShortURL(originalURL)
	err := f.Store.Set(shortURL, originalURL, userID)

	if err != nil {
		return "", err
	}

	result, err := url.JoinPath(f.BaseURL, shortURL)

	if err != nil {
		return "", err
	}

	return result, nil
}

func (f *Facade) GetURLFacade(shortURL string) (storage.URLDetails, error) {
	URLDetails, found := f.Store.Get(shortURL)

	if !found {
		return URLDetails, fmt.Errorf("short URL not found")
	}

	return URLDetails, nil
}

func (f *Facade) APIUserURLFacade(userID string) ([]BatchUserShortenResponse, error) {
	var response []BatchUserShortenResponse

	batch, err := f.Store.GetURLsByUserID(userID)

	if err != nil {
		return response, err
	}

	for _, item := range batch {
		shortURL, err := url.JoinPath(f.BaseURL, item.ShortURL)

		if err != nil {
			return response, err
		}

		resp := BatchUserShortenResponse{
			ShortURL:    shortURL,
			OriginalURL: item.OriginalURL,
		}

		response = append(response, resp)
	}

	return response, nil
}

func (f *Facade) GetUserFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(authenticator.CtxUserKey).(string)

	if !ok {
		return ""
	}

	return userID
}
