package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/memory"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Вспомогательная функция для создания роутера с middleware для тестов
func setupTestRouter() (*gin.Engine, *memory.URLRepository, shortener.URLService) {
	gin.SetMode(gin.TestMode)
	repo := memory.NewURLRepository()
	urlService := shortener.NewURLService(repo)
	router := gin.New()

	// Добавляем middleware для установки user_id в контекст
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Next()
	})

	return router, repo, urlService
}

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
			router, repo, urlService := setupTestRouter()

			// Добавляем тестовые данные если нужно
			if tt.setupData {
				testUserID := "test-user-123"
				err := repo.Store(tt.shortURL, tt.longURL, testUserID)
				require.NoError(t, err)
			}

			// Настройка роутера
			router.GET("/:id", GetHandler(urlService))

			// Создание запроса
			requestURL := "/" + tt.shortURL
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
			router, _, urlService := setupTestRouter()

			// Настройка роутера
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

func TestPostHandlerJSON(t *testing.T) {
	type want struct {
		contentType    string
		statusCode     int
		responsePrefix string
	}
	tests := []struct {
		name    string
		request model.URL
		want    want
	}{
		{
			name:    "Корректный JSON запрос",
			request: model.URL{URL: "https://www.google.com"},
			want: want{
				contentType:    "application/json",
				statusCode:     http.StatusCreated,
				responsePrefix: "http://localhost:8080/",
			},
		},
		{
			name:    "Пустой URL в JSON",
			request: model.URL{URL: ""},
			want: want{
				contentType:    "application/json",
				statusCode:     http.StatusCreated,
				responsePrefix: "http://localhost:8080/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, urlService := setupTestRouter()

			// Настройка роутера
			router.POST("/api/shorten", PostHandlerJSON(urlService, &config.Config{BaseURL: "http://localhost:8080"}))

			// Создание JSON запроса
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")
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

			// Парсим JSON ответ
			var response model.Result
			err = json.Unmarshal(resBody, &response)
			require.NoError(t, err)

			// Проверяем, что ответ начинается с базового URL
			assert.True(t, strings.HasPrefix(response.Result, tt.want.responsePrefix))

			// Проверяем длину ответа (базовый URL + "/" + 6 символов короткой ссылки)
			expectedLength := len(tt.want.responsePrefix) + 6
			assert.Equal(t, expectedLength, len(response.Result))
		})
	}
}

func TestPostHandlerJSON_InvalidJSON(t *testing.T) {
	router, _, urlService := setupTestRouter()
	router.POST("/api/shorten", PostHandlerJSON(urlService, &config.Config{BaseURL: "http://localhost:8080"}))

	// Отправляем невалидный JSON
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	res := w.Result()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	assert.Equal(t, "Неправильное тело запроса", string(resBody))
}

func TestBatchHandler(t *testing.T) {
	tests := []struct {
		name       string
		request    []model.URLBatchRequest
		wantStatus int
		wantLen    int
	}{
		{
			name: "Корректный batch запрос",
			request: []model.URLBatchRequest{
				{CorrelationID: "1", OriginalURL: "https://www.google.com"},
				{CorrelationID: "2", OriginalURL: "https://www.yandex.ru"},
			},
			wantStatus: http.StatusCreated,
			wantLen:    2,
		},
		{
			name:       "Пустой batch",
			request:    []model.URLBatchRequest{},
			wantStatus: http.StatusCreated,
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, urlService := setupTestRouter()
			router.POST("/api/shorten/batch", BatchHandler(urlService, &config.Config{BaseURL: "http://localhost:8080"}))

			// Создание JSON запроса
			requestBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(requestBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Выполнение запроса
			router.ServeHTTP(w, req)
			res := w.Result()

			// Проверки
			assert.Equal(t, tt.wantStatus, res.StatusCode)

			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

			// Парсим JSON ответ
			var response []model.URLBatchResponse
			err = json.Unmarshal(resBody, &response)
			require.NoError(t, err)

			assert.Equal(t, tt.wantLen, len(response))

			// Проверяем каждый элемент ответа
			for i, resp := range response {
				if i < len(tt.request) {
					assert.Equal(t, tt.request[i].CorrelationID, resp.CorrelationID)
					assert.True(t, strings.HasPrefix(resp.ShortURL, "http://localhost:8080/"))
					// Проверяем длину короткой ссылки (базовый URL + "/" + 6 символов)
					expectedLength := len("http://localhost:8080/") + 6
					assert.Equal(t, expectedLength, len(resp.ShortURL))
				}
			}
		})
	}
}

func TestBatchHandler_InvalidJSON(t *testing.T) {
	router, _, urlService := setupTestRouter()
	router.POST("/api/shorten/batch", BatchHandler(urlService, &config.Config{BaseURL: "http://localhost:8080"}))

	// Отправляем невалидный JSON
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	res := w.Result()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)

	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	assert.Equal(t, "Неправильное тело запроса", string(resBody))
}

// Интерфейс для тестирования PingDB
type DBPinger interface {
	PingDB() error
}

// Мок для DBConfig, реализующий интерфейс DBPinger
type mockDBConfig struct {
	shouldFail bool
}

func (m *mockDBConfig) PingDB() error {
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

func TestPingHandler(t *testing.T) {
	tests := []struct {
		name       string
		dbConfig   DBPinger
		wantStatus int
	}{
		{
			name:       "Успешный пинг",
			dbConfig:   &mockDBConfig{shouldFail: false},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Ошибка пинга",
			dbConfig:   &mockDBConfig{shouldFail: true},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router, _, _ := setupTestRouter()

			// Создаем хэндлер с моком
			pingHandler := func(ctx *gin.Context) {
				err := tt.dbConfig.PingDB()
				if err != nil {
					ctx.Status(http.StatusInternalServerError)
					return
				}
				ctx.Status(http.StatusOK)
			}

			router.GET("/ping", pingHandler)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			res := w.Result()
			res.Body.Close()
			assert.Equal(t, tt.wantStatus, res.StatusCode)
		})
	}
}

// func TestGetURlsHahdler(t *testing.T) {

// 	type want struct {
// 		contentType string
// 		statusCode  int
// 		location    string
// 	}
// 	tests := []struct {
// 		name      string
// 		shortURL  string
// 		longURL   string
// 		want      want
// 		setupData bool
// 	}{
// 		{
// 			name:      "Корректный запрос",
// 			shortURL:  "test123",
// 			longURL:   "www.google.com",
// 			setupData: true,
// 			want: want{
// 				contentType: "text/plain",
// 				statusCode:  http.StatusTemporaryRedirect,
// 				location:    "www.google.com",
// 			},
// 		},
// 		{
// 			name:      "Не нашли ссылку",
// 			shortURL:  "notfound",
// 			longURL:   "",
// 			setupData: false,
// 			want: want{
// 				contentType: "text/plain; charset=utf-8",
// 				statusCode:  http.StatusNotFound,
// 				location:    "",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			router, repo, urlService := setupTestRouter()

// 			// Добавляем тестовые данные если нужно
// 			if tt.setupData {
// 				testUserID := "test-user-123"
// 				err := repo.Store(tt.shortURL, tt.longURL, testUserID)
// 				require.NoError(t, err)
// 			}

// 			// Настройка роутера
// 			router.GET("/api/user/urls/:id", GetURlsHandler(urlService))

// 			// Создание запроса
// 			requestURL := "/api/user/urls/" + tt.shortURL
// 			r := httptest.NewRequest(http.MethodGet, requestURL, nil)
// 			w := httptest.NewRecorder()

// 			// Выполнение запроса
// 			router.ServeHTTP(w, r)
// 			res := w.Result()

// 			// Проверки
// 			assert.Equal(t, tt.want.statusCode, res.StatusCode)

// 			defer res.Body.Close()
// 			_, err := io.ReadAll(res.Body)
// 			require.NoError(t, err)

// 			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
// 			assert.Equal(t, tt.want.location, res.Header.Get("Location"))
// 		})
// 	}
// }

// Тест для функции shortenBatch
func TestShortenBatch(t *testing.T) {
	_, _, urlService := setupTestRouter()

	// Создаем контекст для тестирования
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", "test-user-123")

	tests := []struct {
		name    string
		request []model.URLBatchRequest
		wantLen int
		wantErr bool
	}{
		{
			name: "Корректный batch",
			request: []model.URLBatchRequest{
				{CorrelationID: "1", OriginalURL: "https://www.google.com"},
				{CorrelationID: "2", OriginalURL: "https://www.yandex.ru"},
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "Пустой batch",
			request: []model.URLBatchRequest{},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := shortenBatch(tt.request, urlService, "http://localhost:8080", c)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantLen, len(response))

				// Проверяем каждый элемент ответа
				for i, resp := range response {
					if i < len(tt.request) {
						assert.Equal(t, tt.request[i].CorrelationID, resp.CorrelationID)
						assert.True(t, strings.HasPrefix(resp.ShortURL, "http://localhost:8080/"))
					}
				}
			}
		})
	}
}

// Тест для функции handleConflictError
func TestHandleConflictError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		baseURL      string
		wantFullURL  string
		wantConflict bool
	}{
		{
			name:         "Не ошибка конфликта",
			err:          assert.AnError,
			baseURL:      "http://localhost:8080",
			wantFullURL:  "",
			wantConflict: false,
		},
		{
			name:         "Nil ошибка",
			err:          nil,
			baseURL:      "http://localhost:8080",
			wantFullURL:  "",
			wantConflict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullURL, isConflict := handleConflictError(tt.err, tt.baseURL)
			assert.Equal(t, tt.wantConflict, isConflict)
			assert.Equal(t, tt.wantFullURL, fullURL)
		})
	}
}
