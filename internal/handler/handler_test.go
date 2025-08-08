package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHandler(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		response    string
	}
	tests := []struct {
		name      string
		request   string
		shortURLs map[string]string
		want      want
		method    string
	}{
		{
			name:      "Корректный запрос",
			method:    http.MethodGet,
			shortURLs: map[string]string{"wag1oE": "www.google.com"},
			request:   "http://localhost:8080/wag1oE",
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusTemporaryRedirect,
				response:    "www.google.com",
			},
		},
		{
			name:      "Не нашли ссылку",
			method:    http.MethodGet,
			shortURLs: map[string]string{"1111": "www.google.com"},
			request:   "http://localhost:8080/wag1oE",
			want: want{
				contentType: "text/plain; charset=utf-8", // по умолчанию
				statusCode:  http.StatusBadRequest,
				response:    "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.request, nil)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(GetHandler(tt.shortURLs))
			h(w, r)
			res := w.Result()

			// Проверяем коды статуса
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			// Если метод корректный и мы все ок возвращаем

			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			// assert.Equal(t, 28, len(resBody))
			assert.Equal(t, tt.want.response, res.Header.Get("Location"))
		})
	}
}

func TestPostHandler(t *testing.T) {

	type want struct {
		contentType string
		statusCode  int
		response    string
	}
	tests := []struct {
		name      string
		request   string
		shortURLs map[string]string
		want      want
		method    string
	}{
		{
			name:      "Корректный запрос",
			method:    http.MethodPost,
			shortURLs: map[string]string{},
			request:   "www.google.com",
			want: want{
				contentType: "text/plain",
				statusCode:  201,
				response:    "http://localhost:8080/",
			},
		},
		{
			name:      "Не тот метод",
			method:    http.MethodGet,
			shortURLs: map[string]string{},
			request:   "www.google.com",
			want: want{
				contentType: "",
				statusCode:  400,
				response:    "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.request))
			w := httptest.NewRecorder()
			h := http.HandlerFunc(PostHandler(tt.shortURLs))
			h(w, r)
			res := w.Result()

			// Проверяем коды статуса
			assert.Equal(t, tt.want.statusCode, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			// Если метод корректный и мы все ок возвращаем
			if tt.method == http.MethodPost {
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
				assert.Equal(t, 28, len(resBody))
			}

		})
	}
}
