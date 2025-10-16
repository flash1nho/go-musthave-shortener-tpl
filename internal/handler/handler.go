package handler

import (
    "fmt"
    "net/http"
    "io"

    "go-musthave-shortener-tpl/internal/helpers"
    "go-musthave-shortener-tpl/internal/storage"
)

type Handler struct {
    store *storage.Storage
    url string
}

func NewHandler(store *storage.Storage, url string) *Handler {
    return &Handler{
        store: store,
        url: url,
    }
}

func (h *Handler) PostURLHandler(res http.ResponseWriter, req *http.Request) {
    body, err := io.ReadAll(req.Body)

    if err != nil {
        http.Error(res, err.Error(), http.StatusBadRequest)
        return
    }

    defer req.Body.Close()

    originalURL := string(body)

    if originalURL == "" {
        http.Error(res, "body is missing", http.StatusBadRequest)
        return
    }

    shortURL := helpers.GenerateShortURL(originalURL)
    h.store.Set(shortURL, originalURL)

    res.WriteHeader(http.StatusCreated)
    fmt.Fprintf(res, "http://%s/%s", h.url, shortURL)
}

func (h *Handler) GetURLHandler(res http.ResponseWriter, req *http.Request) {
    shortURL := req.URL.Path[1:]

    if shortURL == "" {
        http.Error(res, "id parameter is missing", http.StatusBadRequest)
        return
    }

    originalURL, found := h.store.Get(shortURL)

    if !found {
        http.Error(res, "Short URL not found", http.StatusBadRequest)
        return
    }

    http.Redirect(res, req, originalURL, http.StatusTemporaryRedirect)
}
