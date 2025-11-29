package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/memory"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupInMemoryRouter() (*gin.Engine, *memory.URLRepository) {
	router := gin.New()
	repo := memory.NewURLRepository()

	// Мидлварь для user_id (один раз на роутер)
	userID := "test-user-123"
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("had_cookie", true)
		c.Set("cookie_was_valid", true)
		c.Next()
	})

	return router, repo
}

// =============================================================================
// БЕНЧМАРКИ С IN-MEMORY РЕПОЗИТОРИЕМ
// =============================================================================

func BenchmarkPostHandler_InMemory(b *testing.B) {
	router, repo := setupInMemoryRouter()
	service := shortener.NewURLService(repo)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	auditPub := &audit.Publisher{}

	router.POST("/", PostHandler(service, cfg, auditPub))

	payload := []byte("https://example.com/very/long/url/path")
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req.Body = io.NopCloser(bytes.NewReader(payload))
		req.ContentLength = int64(len(payload))
		router.ServeHTTP(w, req)
		w.Body.Reset() // очищаем буфер
	}
}

func BenchmarkPostHandlerJSON_InMemory(b *testing.B) {
	router, repo := setupInMemoryRouter()
	service := shortener.NewURLService(repo)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	auditPub := &audit.Publisher{}

	router.POST("/api/shorten", PostHandlerJSON(service, cfg, auditPub))

	body := model.URL{URL: "https://example.com/very/long/url/path"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/shorten", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req.Body = io.NopCloser(bytes.NewReader(jsonBody))
		req.ContentLength = int64(len(jsonBody))
		router.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkGetHandler_InMemory(b *testing.B) {
	router, repo := setupInMemoryRouter()
	service := shortener.NewURLService(repo)
	auditPub := &audit.Publisher{}

	router.GET("/:id", GetHandler(service, auditPub))

	// Создаём одну ссылку
	shortURL, _ := service.Shorten("https://benchmark.example.com", "test-user-123")

	req := httptest.NewRequest("GET", "/"+shortURL, nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkBatchHandler_InMemory(b *testing.B) {
	router, repo := setupInMemoryRouter()
	service := shortener.NewURLService(repo)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	router.POST("/api/shorten/batch", BatchHandler(service, cfg))

	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run("Batch"+strconv.Itoa(size), func(b *testing.B) {
			batch := make([]model.URLBatchRequest, size)
			for i := 0; i < size; i++ {
				batch[i] = model.URLBatchRequest{
					CorrelationID: strconv.Itoa(i),
					OriginalURL:   "https://example.com/url/" + strconv.Itoa(i),
				}
			}
			jsonBody, _ := json.Marshal(batch)

			req := httptest.NewRequest("POST", "/api/shorten/batch", nil)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req.Body = io.NopCloser(bytes.NewReader(jsonBody))
				req.ContentLength = int64(len(jsonBody))
				router.ServeHTTP(w, req)
				w.Body.Reset()
			}
		})
	}
}

func BenchmarkGetUserURLsHandler_InMemory(b *testing.B) {
	router, repo := setupInMemoryRouter()
	service := shortener.NewURLService(repo)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	router.GET("/api/user/urls", GetUserURLsHandler(service, cfg))

	// Создаём 100 URL для пользователя
	userID := "test-user-123"
	for i := 0; i < 100; i++ {
		_, _ = service.Shorten("https://example.com/user/"+strconv.Itoa(i), userID)
	}

	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
		w.Body.Reset()
	}
}

func BenchmarkShortenBatch_InMemory(b *testing.B) {
	repo := memory.NewURLRepository()
	service := shortener.NewURLService(repo)
	baseURL := "http://localhost:8080"
	userID := "test-user-123"

	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run("Batch"+strconv.Itoa(size), func(b *testing.B) {
			reqs := make([]model.URLBatchRequest, size)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Генерируем новые URL каждый раз
				for j := 0; j < size; j++ {
					reqs[j] = model.URLBatchRequest{
						CorrelationID: strconv.Itoa(j),
						OriginalURL:   "https://bench.example/" + strconv.Itoa(i*size+j),
					}
				}

				_, err := shortenBatch(reqs, service, baseURL, userID)
				if err != nil {
					b.Fatalf("shortenBatch failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkHandleConflictError_InMemory(b *testing.B) {
	baseURL := "http://localhost:8080"

	b.Run("NoError", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = handleConflictError(nil, baseURL)
		}
	})

	b.Run("WithConflictError", func(b *testing.B) {
		err := ErrURLConflictError{ExistingShortURL: "abc123"}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = handleConflictError(err, baseURL)
		}
	})
}

type ErrURLConflictError struct {
	ExistingShortURL string
}

func (e ErrURLConflictError) Error() string {
	return "conflict: " + e.ExistingShortURL
}
