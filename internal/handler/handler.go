package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/facade"
	"github.com/flash1nho/go-musthave-shortener-tpl/internal/helpers"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// generate:reset
type ShortenRequest struct {
	URL string `json:"url"`
}

// generate:reset
type ShortenResponse struct {
	Result string `json:"result"`
}

// generate:reset
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// generate:reset
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// generate:reset
type Batch struct {
	urlMappings map[string]string
}

// generate:reset
type Handler struct {
	Facade *facade.Facade
	log    *zap.Logger
}

func NewHandler(facade *facade.Facade, settings config.SettingsObject) *Handler {
	return &Handler{
		Facade: facade,
		log:    settings.Log,
	}
}

func (h *Handler) PostURLHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.Facade.GetUserFromContext(r.Context())

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

	result, err := h.Facade.PostURLFacade(userID, originalURL)
	handleStatusConflict(w, err)

	fmt.Fprintln(w, result)
}

func (h *Handler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := r.URL.Path[1:]

	if shortURL == "" {
		http.Error(w, "id parameter is missing", http.StatusBadRequest)
		return
	}

	URLDetails, err := h.Facade.GetURLFacade(shortURL)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if URLDetails.IsDeleted {
		w.WriteHeader(http.StatusGone)
		return
	}

	http.Redirect(w, r, URLDetails.OriginalURL, http.StatusTemporaryRedirect)
}

// APIShortenPostURLHandler - принимает в теле запроса строку URL для сокращения:
//
//	{"url":"<url>"}
//
// Возвращает ответ http.StatusCreated (201) и сокращенный URL в виде JSON:
//
//	{"result":"<shorten_url>"}
//
// @Tags shorten
// @Summary Создает сокращенную ссылку
// @Security Auth
// @ID APIShortenPostURLHandler
// @Accept  json
// @Produce json
// @Success 201
// @Failure 400
// @Failure 401
// @Failure 409
// @Failure 500
// @Router /api/shorten [POST]
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
	err = h.Facade.Store.Set(shortURL, req.URL, "")
	handleStatusConflict(w, err)

	shortURL, _ = url.JoinPath(h.Facade.BaseURL, shortURL)

	response := ShortenResponse{
		Result: shortURL,
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.Facade.Store.Pool == nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		h.log.Error("ошибка пинга базы данных")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// APIShortenBatchPostURLHandler - принимает в теле запроса список строк URL для сокращения:
//
//	[
//	    {
//	        "correlation_id": "<строковый идентификатор>",
//	        "original_url": "<URL для сокращения>"
//	    },
//	    ...
//	]
//
// Возвращает ответ http.StatusCreated (201) и сокращенный URL в виде JSON:
//
//	[
//	    {
//	        "correlation_id": "<строковый идентификатор из объекта запроса>",
//	        "short_url": "<shorten_url>"
//	    },
//	    ...
//	]
//
// @Tags shorten
// @Summary Создает несколько сокращенных ссылок
// @Security Auth
// @ID APIShortenBatchPostURLHandler
// @Accept  json
// @Produce json
// @Success 201
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /api/shorten/batch [POST]
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
		shortURL, err := url.JoinPath(h.Facade.BaseURL, sURL)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			h.log.Error(fmt.Sprintf("Ошибка батчинга: %v", err))
			return
		}

		resp := BatchShortenResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		}

		batch.urlMappings[sURL] = item.OriginalURL

		response = append(response, resp)
	}

	h.Facade.Store.SetBatch(batch.urlMappings)

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) APIUserURLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := h.Facade.GetUserFromContext(r.Context())
	result, err := h.Facade.APIUserURLFacade(userID)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		h.log.Error(fmt.Sprintf("Ошибка получения URLs по user_id: %v", err))
		return
	}

	if len(result) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(result)
}

// APIUserDeleteURLHandler - помечает ссылки пользователя как удаленные.
// Формат запроса:
//
//	[ "a", "b", "c", "d", ...]
//
// Возвращает ответ http.StatusAccepted (202)
//
// @Tags url delete batch
// @Summary Удаляет несколько сокращенных ссылок
// @Security Auth
// @ID APIUserDeleteURLHandler
// @Accept  json
// @Produce json
// @Success 202
// @Failure 400
// @Failure 401
// @Failure 500
// @Router /api/user/urls [DELETE]
func (h *Handler) APIUserDeleteURLHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userID := h.Facade.GetUserFromContext(r.Context())

	if userID != "" {
		var urls []string

		err := json.NewDecoder(r.Body).Decode(&urls)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		err = h.Facade.Store.DeleteBatch(userID, urls)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) APIInternalStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats, err := h.Facade.Store.GetStats()

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
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
