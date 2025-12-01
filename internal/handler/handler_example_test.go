package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/handler"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/memory"
	"github.com/Popolzen/shortener/internal/repository/mocks"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

// setupTestRouter создает роутер для примеров с in-memory репозиторием
func setupTestRouter() (*gin.Engine, shortener.URLService) {
	gin.SetMode(gin.TestMode)
	repo := memory.NewURLRepository()
	urlService := shortener.NewURLService(repo)
	router := gin.New()

	// Middleware для установки user_id
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "example-user-123")
		c.Set("had_cookie", true)
		c.Set("cookie_was_valid", true)
		c.Next()
	})

	return router, urlService
}

// setupTestRouterWithMock создает роутер с mock-репозиторием
func setupTestRouterWithMock(ctrl *gomock.Controller) (*gin.Engine, *mocks.MockURLRepository) {
	gin.SetMode(gin.TestMode)
	repo := mocks.NewMockURLRepository(ctrl)
	router := gin.New()

	// Middleware для установки user_id
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "example-user-123")
		c.Set("had_cookie", true)
		c.Set("cookie_was_valid", true)
		c.Next()
	})

	return router, repo
}

// ExamplePostHandler демонстрирует создание короткой ссылки через POST запрос
func ExamplePostHandler() {
	router, urlService := setupTestRouter()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	pub := audit.NewPublisher()

	router.POST("/", handler.PostHandler(urlService, cfg, pub))

	// Создаем запрос с длинным URL
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Short URL format:", strings.HasPrefix(string(body), "http://localhost:8080/"))
	// Output:
	// Status: 201
	// Short URL format: true
}

// ExampleGetHandler демонстрирует получение оригинального URL по короткой ссылке
func ExampleGetHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	router, mockRepo := setupTestRouterWithMock(ctrl)
	pub := audit.NewPublisher()
	urlService := shortener.NewURLService(mockRepo)

	// Настраиваем mock: возвращаем оригинальный URL
	mockRepo.EXPECT().Get("abc123").Return("https://example.com", nil)

	router.GET("/:id", handler.GetHandler(urlService, pub))

	// Создаем запрос для получения URL
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Location header:", resp.Header.Get("Location"))
	// Output:
	// Status: 307
	// Location header: https://example.com
}

// ExamplePostHandlerJSON демонстрирует создание короткой ссылки через JSON API
func ExamplePostHandlerJSON() {
	router, urlService := setupTestRouter()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	pub := audit.NewPublisher()

	router.POST("/api/shorten", handler.PostHandlerJSON(urlService, cfg, pub))

	// Создаем JSON запрос
	requestBody := map[string]string{"url": "https://example.com"}
	jsonData, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result map[string]string
	json.Unmarshal(body, &result)

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Has result field:", result["result"] != "")
	// Output:
	// Status: 201
	// Has result field: true
}

// ExampleBatchHandler демонстрирует пакетное создание коротких ссылок
func ExampleBatchHandler() {
	router, urlService := setupTestRouter()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	router.POST("/api/shorten/batch", handler.BatchHandler(urlService, cfg))

	// Создаем батч запрос
	batch := []map[string]string{
		{"correlation_id": "1", "original_url": "https://example.com"},
		{"correlation_id": "2", "original_url": "https://google.com"},
	}
	jsonData, _ := json.Marshal(batch)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result []map[string]string
	json.Unmarshal(body, &result)

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Batch size:", len(result))
	fmt.Println("Has correlation_id:", result[0]["correlation_id"] != "")
	// Output:
	// Status: 201
	// Batch size: 2
	// Has correlation_id: true
}

// ExampleGetUserURLsHandler демонстрирует получение всех URL пользователя
func ExampleGetUserURLsHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	router, mockRepo := setupTestRouterWithMock(ctrl)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlService := shortener.NewURLService(mockRepo)

	// Настраиваем mock: возвращаем список URL пользователя
	mockRepo.EXPECT().GetUserURLs("example-user-123").Return([]model.URLPair{
		{ShortURL: "abc123", OriginalURL: "https://example1.com"},
		{ShortURL: "def456", OriginalURL: "https://example2.com"},
	}, nil)

	router.GET("/api/user/urls", handler.GetUserURLsHandler(urlService, cfg))

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var urls []model.URLPair
	json.Unmarshal(body, &urls)

	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("URL count:", len(urls))
	fmt.Println("First URL contains base:", strings.HasPrefix(urls[0].ShortURL, "http://localhost:8080/"))
	// Output:
	// Status: 200
	// URL count: 2
	// First URL contains base: true
}

// ExampleDeleteURLsHandler демонстрирует асинхронное удаление URL
func ExampleDeleteURLsHandler() {
	ctrl := gomock.NewController(nil)
	defer ctrl.Finish()

	router, mockRepo := setupTestRouterWithMock(ctrl)
	urlService := shortener.NewURLService(mockRepo)

	// Настраиваем mock: ожидаем вызов DeleteURLs
	mockRepo.EXPECT().DeleteURLs("example-user-123", []string{"url1", "url2", "url3"})

	router.DELETE("/api/user/urls", handler.DeleteURLsHandler(urlService))

	// Создаем запрос на удаление
	urlsToDelete := []string{"url1", "url2", "url3"}
	jsonData, _ := json.Marshal(urlsToDelete)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	fmt.Println("Status:", resp.StatusCode)
	// Output:
	// Status: 202
}
