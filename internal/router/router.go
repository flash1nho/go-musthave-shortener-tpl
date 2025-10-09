package router

import (
    "fmt"
    "net/http"
    "sync"
    "io"

    "go-musthave-shortener-tpl/internal/helpers"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

var urlStore = struct {
    sync.RWMutex
    m map[string]string
}{m: make(map[string]string)}

func init() {
    if urlStore.m == nil {
        urlStore.m = make(map[string]string)
    }
}

type postHandlerStruct struct {
    url string
}

func Start(a string, b string) {
    if a == b {
        serverA := chi.NewRouter()
        serverA.Use(middleware.Logger)
        handler := postHandlerStruct{url: a}
        serverA.Post("/", handler.postURLHandler)
        serverA.Get("/{id}", getURLHandler)

        err := http.ListenAndServe(a, serverA)
        if err != nil {
            panic(err)
        }
    } else {
        serverA := chi.NewRouter()
        serverA.Use(middleware.Logger)
        handler := postHandlerStruct{url: b}
        serverA.Post("/", handler.postURLHandler)

        serverB := chi.NewRouter()
        serverB.Use(middleware.Logger)
        serverB.Get("/{id}", getURLHandler)

        go func() {
            err := http.ListenAndServe(a, serverA)
            if err != nil {
                panic(err)
            }
        }()

        err := http.ListenAndServe(b, serverB)
        if err != nil {
            panic(err)
        }
    }
}

func (b postHandlerStruct) postURLHandler(res http.ResponseWriter, req *http.Request) {
    if req.Method != http.MethodPost {
        http.Error(res, "Method not allowed", http.StatusBadRequest)
        return
    }

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

    urlStore.Lock()
    urlStore.m[shortURL] = originalURL
    urlStore.Unlock()

    res.WriteHeader(http.StatusCreated)
    fmt.Fprintf(res, "http://%s/%s", b.url, shortURL)
}

func getURLHandler(res http.ResponseWriter, req *http.Request) {
    if req.Method != http.MethodGet {
        http.Error(res, "Method not allowed", http.StatusBadRequest)
        return
    }

    shortURL := req.URL.Path[1:]
    if shortURL == "" {
        http.Error(res, "id parameter is missing", http.StatusBadRequest)
        return
    }

    urlStore.RLock()
    originalURL, found := urlStore.m[shortURL]
    urlStore.RUnlock()

    if !found {
        http.Error(res, "Short URL not found", http.StatusBadRequest)
        return
    }

    http.Redirect(res, req, originalURL, http.StatusTemporaryRedirect)
}
