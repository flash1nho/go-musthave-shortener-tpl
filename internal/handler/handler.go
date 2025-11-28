package handler

import (
    "fmt"
    "net/http"
    "net/url"
    "io"
    "encoding/json"
    "errors"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/helpers"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"

    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgerrcode"
    "go.uber.org/zap"
)

type ShortenRequest struct {
    URL string `json:"url"`
}

type ShortenResponse struct {
    Result string `json:"result"`
}

type BatchShortenRequest struct {
    CorrelationID string `json:"correlation_id"`
    OriginalURL   string `json:"original_url"`
}

type BatchShortenResponse struct {
    CorrelationID string `json:"correlation_id"`
    ShortURL      string `json:"short_url"`
}

type BatchUserShortenResponse struct {
    ShortURL    string `json:"short_url"`
    OriginalURL string `json:"original_url"`
}

type Batch struct {
    urlMappings map[string]string
}

type Handler struct {
    store  *storage.Storage
    server config.Server
    log    *zap.Logger
}

func NewHandler(store *storage.Storage, server config.Server, log *zap.Logger) *Handler {
    return &Handler{
        store:  store,
        server: server,
        log:    log,
    }
}

func (h *Handler) PostURLHandler(w http.ResponseWriter, r *http.Request) {
    userID, err := helpers.GetUserIDFromCookie(r)

    if err != nil || userID == "" {
        userID, err = helpers.GenerateUniqueUserID()

        if err != nil {
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }

        if err := helpers.SetSignedCookie(w, userID); err != nil {
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        }
    }

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
    err = h.store.Set(shortURL, originalURL, userID)
    handleStatusConflict(w, err)

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
    w.Header().Set("Content-Type", "application/json")

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
    err = h.store.Set(shortURL, req.URL, "")
    handleStatusConflict(w, err)

    shortURL, _ = url.JoinPath(h.server.BaseURL, shortURL)

    response := ShortenResponse{
        Result: shortURL,
    }

    json.NewEncoder(w).Encode(response)
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
    if h.store.Pool == nil {
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        h.log.Fatal("ошибка пинга базы данных")
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (h *Handler) APIShortenBatchPostURLHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    var req []BatchShortenRequest

    err := json.NewDecoder(r.Body).Decode(&req)

    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if len(req) == 0 {
        http.Error(w, "body is missing", http.StatusBadRequest)
        return
    }

    var response []BatchShortenResponse

    batch := &Batch{
        urlMappings: make(map[string]string),
    }

    for _, item := range req {
        sURL := helpers.GenerateShortURL(item.OriginalURL)
        shortURL, err := url.JoinPath(h.server.BaseURL, sURL)

        if err != nil {
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            h.log.Fatal(fmt.Sprintf("Ошибка батчинга: %v", err))
            return
        }

        resp := BatchShortenResponse{
          CorrelationID: item.CorrelationID,
          ShortURL: shortURL,
        }

        batch.urlMappings[sURL] = item.OriginalURL

        response = append(response, resp)
    }

    h.store.SetBatch(batch.urlMappings)

    w.WriteHeader(http.StatusCreated)

    json.NewEncoder(w).Encode(response)
}

func (h *Handler) APIUserURLHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    userID, err := helpers.GetUserIDFromCookie(r)

    if err != nil || userID == "" {
        if err := helpers.SetSignedCookie(w, userID); err != nil {
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            return
        } else {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
        return
    }

    var response []BatchUserShortenResponse

    batch, err := h.store.GetURLsByUserID(userID)

    if err != nil {
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        h.log.Fatal(fmt.Sprintf("Ошибка получения URLs по user_id: %v", err))
        return
    }

    for sURL, originalURL := range batch {
        shortURL, err := url.JoinPath(h.server.BaseURL, sURL)

        if err != nil {
            http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            h.log.Fatal(fmt.Sprintf("Ошибка батчинга: %v", err))
            return
        }

        resp := BatchUserShortenResponse{
          ShortURL: shortURL,
          OriginalURL: originalURL,
        }

        response = append(response, resp)
    }

    w.WriteHeader(http.StatusOK)

    json.NewEncoder(w).Encode(response)
}

func handleStatusConflict(w http.ResponseWriter, err error) {
    if err != nil {
        var pgErr *pgconn.PgError

        if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
            w.WriteHeader(http.StatusConflict)
        } else {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }
    } else {
        w.WriteHeader(http.StatusCreated)
    }
}
