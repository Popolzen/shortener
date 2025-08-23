package main

import (
	"net/http"

	"github.com/Popolzen/shortener/internal/handler"
)

func main() {
	shortURLs := make(map[string]string)

	mux := http.NewServeMux()
	mux.Handle("/", handler.PostHandler(shortURLs))
	mux.Handle("/{id}", handler.GetHandler(shortURLs))

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
