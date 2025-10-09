package router

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestPostURLHandler(t *testing.T) {
    handler := &postHandlerStruct{url: "localhost:8080"}

    // описываем набор данных: метод запроса, ожидаемый код ответа, тело ответа, тело запроса
    testCases := []struct {
        method       string
        status int
        responseBody string
        requestBody string
    }{
        {method: http.MethodGet, status: http.StatusBadRequest, responseBody: "Method not allowed", requestBody: ""},
        {method: http.MethodPut, status: http.StatusBadRequest, responseBody: "Method not allowed", requestBody: ""},
        {method: http.MethodDelete, status: http.StatusBadRequest, responseBody: "Method not allowed", requestBody: ""},
        {method: http.MethodPost, status: http.StatusBadRequest, responseBody: "body is missing", requestBody: ""},
        {method: http.MethodPost, status: http.StatusCreated, responseBody: "http://localhost:8080/ipkjUVtE", requestBody: "https://practicum.yandex.ru"},
    }

    for _, tc := range testCases {
        t.Run(tc.method, func(t *testing.T) {
            r := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.requestBody))
            w := httptest.NewRecorder()

            // вызовем хендлер как обычную функцию, без запуска самого сервера
            handler.postURLHandler(w, r)

            assert.Equal(t, tc.status, w.Code, "Код ответа не совпадает с ожидаемым")
            // проверим корректность полученного тела ответа, если мы его ожидаем
            if tc.responseBody != "" {
                // assert.JSONEq помогает сравнить две JSON-строки
                assert.Equal(t, tc.responseBody, strings.TrimSuffix(w.Body.String(), "\n"), "Тело ответа не совпадает с ожидаемым")
            }
        })
    }
}

func TestGetURLHandler(t *testing.T) {
    // описываем набор данных: метод запроса, ожидаемый код ответа, тело ответа, path запроса
    testCases := []struct {
        method       string
        status int
        responseBody string
        path string
    }{
        {method: http.MethodPost, status: http.StatusBadRequest, responseBody: "Method not allowed", path: "/"},
        {method: http.MethodPut, status: http.StatusBadRequest, responseBody: "Method not allowed", path: "/"},
        {method: http.MethodDelete, status: http.StatusBadRequest, responseBody: "Method not allowed", path: "/"},
        {method: http.MethodGet, status: http.StatusBadRequest, responseBody: "id parameter is missing", path: "/"},
        {method: http.MethodGet, status: http.StatusBadRequest, responseBody: "Short URL not found", path: "/abcdefgh"},
        {method: http.MethodGet, status: http.StatusTemporaryRedirect, responseBody: "", path: "/ipkjUVtE"},
    }

    for _, tc := range testCases {
        t.Run(tc.method, func(t *testing.T) {
            r := httptest.NewRequest(tc.method, tc.path, nil)
            w := httptest.NewRecorder()

            // вызовем хендлер как обычную функцию, без запуска самого сервера
            getURLHandler(w, r)

            assert.Equal(t, tc.status, w.Code, "Код ответа не совпадает с ожидаемым")
            // проверим корректность полученного тела ответа, если мы его ожидаем
            if tc.responseBody != "" {
                // assert.JSONEq помогает сравнить две JSON-строки
                assert.Equal(t, tc.responseBody, strings.TrimSuffix(w.Body.String(), "\n"), "Тело ответа не совпадает с ожидаемым")
            }
        })
    }
}
