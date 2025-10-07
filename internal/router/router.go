package router

import (
    "fmt"
    "net/http"
    "sync"
    "io"
    "go-musthave-shortener-tpl/internal/helpers"
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

func Start() {
    router := http.NewServeMux()
    router.HandleFunc("/", postURLHandler)
    router.HandleFunc("/{id}", getURLHandler)

    err := http.ListenAndServe(":8080", router)
    if err != nil {
        panic(err)
    }
}

func postURLHandler(res http.ResponseWriter, req *http.Request) {
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
    fmt.Fprintf(res, "http://localhost:8080/%s", shortURL)
}

func getURLHandler(res http.ResponseWriter, req *http.Request) {
    if req.Method != http.MethodGet {
        http.Error(res, "Method not allowed", http.StatusBadRequest)
        return
    }

    shortURL := req.PathValue("id")
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
