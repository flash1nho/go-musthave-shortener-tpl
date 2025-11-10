package handler

import (
    "fmt"
    "net/http"
    "io"
    "encoding/json"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/helpers"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/db"
)

type ShortenRequest struct {
    URL string `json:"url"`
}

type ShortenResponse struct {
    Result string `json:"result"`
}

type Handler struct {
    store *storage.FileStorage
    server config.Server
    databaseDSN string
}

func NewHandler(store *storage.FileStorage, server config.Server, databaseDSN string) *Handler {
    return &Handler{
        store: store,
        server: server,
        databaseDSN: databaseDSN,
    }
}

func (h *Handler) PostURLHandler(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)

    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    defer r.Body.Close()

    originalURL := string(body)

    if originalURL == "" {
        http.Error(w, "body is missing", http.StatusBadRequest)
        return
    }

    shortURL := helpers.GenerateShortURL(originalURL)
    h.store.Set(shortURL, originalURL)

    w.WriteHeader(http.StatusCreated)
    fmt.Fprintf(w, "%s/%s", h.server.BaseURL, shortURL)
}

func (h *Handler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
    shortURL := r.URL.Path[1:]

    if shortURL == "" {
        http.Error(w, "id parameter is missing", http.StatusBadRequest)
        return
    }

    originalURL, found := h.store.Get(shortURL)

    if !found {
        http.Error(w, "Short URL not found", http.StatusBadRequest)
        return
    }

    http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func (h *Handler) APIShortenPostURLHandler(w http.ResponseWriter, r *http.Request) {
    var req ShortenRequest

    err := json.NewDecoder(r.Body).Decode(&req)

    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if req.URL == "" {
        http.Error(w, "body is missing", http.StatusBadRequest)
        return
    }

    shortURL := helpers.GenerateShortURL(req.URL)
    h.store.Set(shortURL, req.URL)

    response := ShortenResponse{
        Result: h.server.BaseURL + "/" + shortURL,
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)

    json.NewEncoder(w).Encode(response)
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
    _, err := db.Connect(h.databaseDSN)

    if err != nil {
        fmt.Println(err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
