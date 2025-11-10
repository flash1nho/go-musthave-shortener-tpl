package handler

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "encoding/json"
    "testing"

    "github.com/flash1nho/go-musthave-shortener-tpl/internal/config"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/storage"
    "github.com/flash1nho/go-musthave-shortener-tpl/internal/helpers"

    "github.com/stretchr/testify/assert"
)

func testData() (h *Handler, originalURL string, shortURL string) {
    store, _ := storage.NewFileStorage(config.DefaultFilePath)
    h = NewHandler(store, config.ServerData(config.DefaultURL), config.DefaultDatabaseDSN)
    originalURL = "https://practicum.yandex.ru"
    shortURL = helpers.GenerateShortURL(originalURL)

    return h, originalURL, shortURL
}

func TestPostURLHandler(t *testing.T) {
    h, originalURL, shortURL := testData()
    shortURL = h.server.BaseURL + "/" + shortURL

    // описываем набор данных: метод запроса, ожидаемый код ответа, тело ответа, тело запроса
    testCases := []struct {
        method string
        status int
        responseBody string
        requestBody string
    }{
        {method: http.MethodPost, status: http.StatusBadRequest, responseBody: "body is missing", requestBody: ""},
        {method: http.MethodPost, status: http.StatusCreated, responseBody: shortURL, requestBody: originalURL},
    }

    for _, tc := range testCases {
        t.Run(tc.method, func(t *testing.T) {
            r := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.requestBody))
            w := httptest.NewRecorder()

            // вызовем хендлер как обычную функцию, без запуска самого сервера
            h.PostURLHandler(w, r)

            assert.Equal(t, tc.status, w.Code, "Код ответа не совпадает с ожидаемым")
            // проверим корректность полученного тела ответа, если мы его ожидаем
            if tc.responseBody != "" {
                assert.Equal(t, tc.responseBody, strings.TrimSuffix(w.Body.String(), "\n"), "Тело ответа не совпадает с ожидаемым")
            }
        })
    }
}

func TestGetURLHandler(t *testing.T) {
    h, originalURL, shortURL := testData()
    h.store.Set(shortURL, originalURL)

    // описываем набор данных: метод запроса, ожидаемый код ответа, тело ответа, path запроса
    testCases := []struct {
        method string
        status int
        responseBody string
        path string
    }{
        {method: http.MethodGet, status: http.StatusBadRequest, responseBody: "id parameter is missing", path: "/"},
        {method: http.MethodGet, status: http.StatusBadRequest, responseBody: "Short URL not found", path: "/short_url_not_found"},
        {method: http.MethodGet, status: http.StatusTemporaryRedirect, responseBody: "", path: "/" + shortURL},
    }

    for _, tc := range testCases {
        t.Run(tc.method, func(t *testing.T) {
            r := httptest.NewRequest(tc.method, tc.path, nil)
            w := httptest.NewRecorder()

            // вызовем хендлер как обычную функцию, без запуска самого сервера
            h.GetURLHandler(w, r)

            assert.Equal(t, tc.status, w.Code, "Код ответа не совпадает с ожидаемым")
            // проверим корректность полученного тела ответа, если мы его ожидаем
            if tc.responseBody != "" {
                assert.Equal(t, tc.responseBody, strings.TrimSuffix(w.Body.String(), "\n"), "Тело ответа не совпадает с ожидаемым")
            }
        })
    }
}

func TestAPIShortenPostURLHandler(t *testing.T) {
    h, originalURL, shortURL := testData()
    shortURL = h.server.BaseURL + "/" + shortURL

    requestData := ShortenRequest{
        URL: originalURL,
    }
    requestJSONBytes, _ := json.Marshal(requestData)
    requestBody := string(requestJSONBytes)

    responseData := ShortenResponse{
        Result: shortURL,
    }
    responseJSONBytes, _ := json.Marshal(responseData)
    responseBody := string(responseJSONBytes)

    // описываем набор данных: метод запроса, ожидаемый код ответа, тело ответа, тело запроса
    testCases := []struct {
        method string
        status int
        responseBody string
        requestBody string
    }{
        {method: http.MethodPost, status: http.StatusBadRequest, responseBody: "Invalid request body", requestBody: ""},
        {method: http.MethodPost, status: http.StatusBadRequest, responseBody: "body is missing", requestBody: `{"result":""}`},
        {method: http.MethodPost, status: http.StatusCreated, responseBody: responseBody, requestBody: requestBody},
    }

    for _, tc := range testCases {
        t.Run(tc.method, func(t *testing.T) {
            r := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.requestBody))
            w := httptest.NewRecorder()

            // вызовем хендлер как обычную функцию, без запуска самого сервера
            h.APIShortenPostURLHandler(w, r)

            assert.Equal(t, tc.status, w.Code, "Код ответа не совпадает с ожидаемым")
            // проверим корректность полученного тела ответа, если мы его ожидаем
            if tc.responseBody != "" {
                assert.Equal(t, tc.responseBody, strings.TrimSuffix(w.Body.String(), "\n"), "Тело ответа не совпадает с ожидаемым")
            }
        })
    }
}
