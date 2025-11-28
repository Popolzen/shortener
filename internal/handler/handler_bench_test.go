package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/database"
	"github.com/Popolzen/shortener/internal/repository/memory"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func init() {
	gin.SetMode(gin.TestMode)
}

var (
	benchDB   *sql.DB
	benchRepo *database.URLRepository
	setupOnce sync.Once
)

// setupBenchDB настраивает PostgreSQL один раз для всех бенчмарков
func setupBenchDB(b *testing.B) {
	setupOnce.Do(func() {
		ctx := context.Background()

		pgContainer, err := postgres.Run(ctx,
			"postgres:15-alpine",
			postgres.WithDatabase("benchdb"),
			postgres.WithUsername("bench"),
			postgres.WithPassword("bench"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second),
			),
		)
		if err != nil {
			b.Fatalf("Failed to start postgres: %v", err)
		}

		connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			b.Fatalf("Failed to get connection string: %v", err)
		}

		benchDB, err = sql.Open("pgx", connStr)
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}

		// Создаём схему
		_, err = benchDB.Exec(`
			CREATE TABLE IF NOT EXISTS shortened_urls (
				id BIGSERIAL PRIMARY KEY,
				user_id UUID NOT NULL,
				long_url TEXT UNIQUE NOT NULL,
				short_url VARCHAR(20) UNIQUE NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				is_deleted BOOL DEFAULT FALSE,
				CONSTRAINT chk_short_url_length CHECK (length(short_url) >= 4)
			);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_shortened_urls_short_url ON shortened_urls(short_url);
			CREATE INDEX IF NOT EXISTS idx_shortened_urls_user_id ON shortened_urls(user_id);
		`)
		if err != nil {
			b.Fatalf("Failed to create schema: %v", err)
		}

		benchRepo = database.NewURLRepository(benchDB)
	})
}

// setupBenchRouter создает роутер для бенчмарков
func setupBenchRouter(b *testing.B) (*gin.Engine, shortener.URLService, *config.Config, *audit.Publisher) {
	setupBenchDB(b)
	urlService := shortener.NewURLService(benchRepo)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	audit := &audit.Publisher{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
		c.Set("had_cookie", true)
		c.Set("cookie_was_valid", true)
		c.Next()
	})

	return router, urlService, cfg, audit
}

// BenchmarkPostHandler измеряет производительность создания короткой ссылки
func BenchmarkPostHandler(b *testing.B) {
	router, urlService, cfg, audit := setupBenchRouter(b)
	router.POST("/", PostHandler(urlService, cfg, audit))

	body := "https://example.com/very/long/url/path"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkGetHandler измеряет производительность редиректа
func BenchmarkGetHandler(b *testing.B) {
	router, urlService, _, audit := setupBenchRouter(b)
	router.GET("/:id", GetHandler(urlService, audit))

	// Подготовка: создаем тестовую ссылку
	shortURL, _ := urlService.Shorten("https://example.com", "bench-user-123")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/"+shortURL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkPostHandlerJSON измеряет производительность JSON endpoint
func BenchmarkPostHandlerJSON(b *testing.B) {
	router, urlService, cfg, audit := setupBenchRouter(b)
	router.POST("/api/shorten", PostHandlerJSON(urlService, cfg, audit))

	requestBody := model.URL{URL: "https://example.com/very/long/url/path"}
	jsonBody, _ := json.Marshal(requestBody)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/shorten", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkBatchHandler измеряет производительность batch endpoint
func BenchmarkBatchHandler(b *testing.B) {
	router, urlService, cfg, _ := setupBenchRouter(b)
	router.POST("/api/shorten/batch", BatchHandler(urlService, cfg))

	b.Run("Batch10", func(b *testing.B) {
		batch := make([]model.URLBatchRequest, 10)
		for i := 0; i < 10; i++ {
			batch[i] = model.URLBatchRequest{
				CorrelationID: string(rune(i)),
				OriginalURL:   "https://example.com/" + string(rune(i)),
			}
		}
		jsonBody, _ := json.Marshal(batch)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})

	b.Run("Batch100", func(b *testing.B) {
		batch := make([]model.URLBatchRequest, 100)
		for i := 0; i < 100; i++ {
			batch[i] = model.URLBatchRequest{
				CorrelationID: string(rune(i)),
				OriginalURL:   "https://example.com/" + string(rune(i)),
			}
		}
		jsonBody, _ := json.Marshal(batch)

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/api/shorten/batch", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkGetUserURLsHandler измеряет производительность получения пользовательских URL
func BenchmarkGetUserURLsHandler(b *testing.B) {
	router, urlService, cfg, _ := setupBenchRouter(b)

	// Добавляем заглушку авторизации — она должна ставить те же ключи,
	// что и настоящий AuthMiddleware: had_cookie и cookie_was_valid
	router.Use(func(c *gin.Context) {
		c.Set("had_cookie", true)         // была кука
		c.Set("cookie_was_valid", true)   // и она валидная
		c.Set("userID", "bench-user-123") // то, что потом достаёт getUserID
		c.Next()
	})

	router.GET("/api/user/urls", GetUserURLsHandler(urlService, cfg))

	// Подготовка: создаём 50 URL для пользователя
	userID := "bench-user-123"
	for i := 0; i < 50; i++ {
		_, _ = urlService.Shorten("https://example.com/long-url-"+strconv.Itoa(i), userID)
	}

	b.ReportAllocs()
	b.ResetTimer()

	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	// Можно даже куку добавить для красоты, но не обязательно
	req.AddCookie(&http.Cookie{Name: "user", Value: "some-jwt-or-session"})

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkShortenBatch измеряет производительность функции shortenBatch
func BenchmarkShortenBatch(b *testing.B) {
	repo := memory.NewURLRepository()
	urlService := shortener.NewURLService(repo)
	baseURL := "http://localhost:8080"
	userID := "bench-user"

	b.Run("Batch10", func(b *testing.B) {
		requests := make([]model.URLBatchRequest, 10)
		for i := 0; i < 10; i++ {
			requests[i] = model.URLBatchRequest{
				CorrelationID: string(rune(i)),
				OriginalURL:   "https://example.com/" + string(rune(i)),
			}
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = shortenBatch(requests, urlService, baseURL, userID)
		}
	})

	b.Run("Batch100", func(b *testing.B) {
		requests := make([]model.URLBatchRequest, 100)
		for i := 0; i < 100; i++ {
			requests[i] = model.URLBatchRequest{
				CorrelationID: string(rune(i)),
				OriginalURL:   "https://example.com/" + string(rune(i)),
			}
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = shortenBatch(requests, urlService, baseURL, userID)
		}
	})
}

// BenchmarkHandleConflictError измеряет обработку ошибок конфликта
func BenchmarkHandleConflictError(b *testing.B) {
	baseURL := "http://localhost:8080"

	b.Run("NoError", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = handleConflictError(nil, baseURL)
		}
	})

	b.Run("WithConflictError", func(b *testing.B) {
		// Используем правильный импорт из database пакета
		repo := memory.NewURLRepository()
		urlService := shortener.NewURLService(repo)

		// Создаем конфликтную ситуацию
		_, _ = urlService.Shorten("https://example.com/conflict", "user1")

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Пытаемся создать ту же URL снова
			_, err := urlService.Shorten("https://example.com/conflict", "user2")
			_, _ = handleConflictError(err, baseURL)
		}
	})
}
