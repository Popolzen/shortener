package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/repository/memory"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetHandler(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
		location    string
	}
	tests := []struct {
		name      string
		shortURL  string
		longURL   string
		want      want
		method    string
		setupData bool // нужно ли предварительно добавить данные
	}{
		{
			name:      "Корректный запрос",
			method:    http.MethodGet,
			shortURL:  "test123",
			longURL:   "www.google.com",
			setupData: true,
			want: want{
				contentType: "text/plain",
				statusCode:  http.StatusTemporaryRedirect,
				location:    "www.google.com",
			},
		},
		{
			name:      "Не нашли ссылку",
			method:    http.MethodGet,
			shortURL:  "notfound",
			longURL:   "",
			setupData: false,
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusNotFound,
				location:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем реальный memory репозиторий
			repo := memory.NewURLRepository()
			urlService := shortener.NewURLService(repo)

			// Добавляем тестовые данные если нужно
			if tt.setupData {
				err := repo.Store(tt.shortURL, tt.longURL)
				require.NoError(t, err)
			}

			// Настройка роутера
			router := gin.New()
			router.GET("/:id", GetHandler(urlService))

			// Создание запроса
			requestURL := "http://localhost:8080/" + tt.shortURL
			r := httptest.NewRequest(tt.method, requestURL, nil)
			w := httptest.NewRecorder()

			// Выполнение запроса
			router.ServeHTTP(w, r)
			res := w.Result()

			// Проверки
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			defer res.Body.Close()
			_, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, tt.want.location, res.Header.Get("Location"))
		})
	}
}

func TestPostHandler(t *testing.T) {
	type want struct {
		contentType    string
		statusCode     int
		responsePrefix string
	}
	tests := []struct {
		name    string
		request string
		want    want
		method  string
	}{
		{
			name:    "Корректный запрос",
			method:  http.MethodPost,
			request: "www.google.com",
			want: want{
				contentType:    "text/plain",
				statusCode:     http.StatusCreated,
				responsePrefix: "http://localhost:8080/",
			},
		},
		{
			name:    "Пустое тело запроса",
			method:  http.MethodPost,
			request: "",
			want: want{
				contentType:    "text/plain",
				statusCode:     http.StatusCreated,
				responsePrefix: "http://localhost:8080/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем реальный memory репозиторий
			repo := memory.NewURLRepository()
			urlService := shortener.NewURLService(repo)

			// Настройка роутера
			router := gin.New()
			router.POST("/", PostHandler(urlService, &config.Config{BaseURL: "http://localhost:8080"}))

			// Создание запроса
			req := httptest.NewRequest(tt.method, "/", strings.NewReader(tt.request))
			w := httptest.NewRecorder()

			// Выполнение запроса
			router.ServeHTTP(w, req)
			res := w.Result()

			// Проверки
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))

			// Проверяем, что ответ начинается с базового URL
			responseString := string(resBody)
			assert.True(t, strings.HasPrefix(responseString, tt.want.responsePrefix))

			// Проверяем длину ответа (базовый URL + "/" + 6 символов короткой ссылки)
			expectedLength := len(tt.want.responsePrefix) + 6
			assert.Equal(t, expectedLength, len(responseString))
		})
	}
}
