package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Popolzen/shortener/internal/service"
)

func PostHandler(shortURLs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Неподдерживаемый метод", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(r.Body)

		if err != nil {
			http.Error(w, "Неправильное тело запроса", http.StatusBadRequest)
			return
		}

		shortURL, err := service.Shortener(string(body), shortURLs)
		if err != nil {
			http.Error(w, "Не удалось сгенерить короткую ссылку", http.StatusBadRequest)
			return
		}
		shortURLs[shortURL] = string(body)

		shortURL = "http://localhost:8080/" + shortURL

		l := strconv.Itoa(len(shortURL))

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", l)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(shortURL))
	}

}
func GetHandler(shortURLs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Неподдерживаемый метод", http.StatusBadRequest)
			return
		}
		shortURL := strings.TrimPrefix(r.URL.Path, "/")
		if longURL, exists := shortURLs[shortURL]; exists {
			w.Header().Set("Location", longURL)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}

		http.Error(w, "Не нашли ссылку", http.StatusBadRequest)
	}
}
