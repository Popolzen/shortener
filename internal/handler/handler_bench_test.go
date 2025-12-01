// benchmarks_test.go
package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/database"
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

		_, err = benchDB.Exec(`
			CREATE TABLE IF NOT EXISTS shortened_urls (
				id BIGSERIAL PRIMARY KEY,
				user_id UUID NOT NULL,
				long_url TEXT UNIQUE NOT NULL,
				short_url VARCHAR(20) UNIQUE NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				is_deleted BOOL DEFAULT FALSE
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

func setupBenchRouter(b *testing.B) (*gin.Engine, *database.URLRepository) {
	setupBenchDB(b)
	router := gin.New()

	userID := "550e8400-e29b-41d4-a716-446655440000"

	router.Use(func(c *gin.Context) {

		c.Set("user_id", userID)
		c.Set("had_cookie", true)
		c.Set("cookie_was_valid", true)
		c.Next()
	})

	return router, benchRepo
}

func BenchmarkPostHandler(b *testing.B) {
	router, repo := setupBenchRouter(b)
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
	}
}

func BenchmarkPostHandlerJSON(b *testing.B) {
	router, repo := setupBenchRouter(b)
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
	}
}

func BenchmarkBatchHandler(b *testing.B) {
	router, repo := setupBenchRouter(b)
	service := shortener.NewURLService(repo)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	router.POST("/api/shorten/batch", BatchHandler(service, cfg))

	sizes := []struct {
		name string
		n    int
	}{
		{"10", 10}, {"50", 50}, {"100", 100}, {"500", 500},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			batch := make([]model.URLBatchRequest, s.n)
			for i := 0; i < s.n; i++ {
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
			}
		})
	}
}

func BenchmarkGetHandler(b *testing.B) {
	router, repo := setupBenchRouter(b)
	service := shortener.NewURLService(repo)
	auditPub := &audit.Publisher{}

	router.GET("/:id", GetHandler(service, auditPub))

	// Создаём одну ссылку
	shortURL, _ := service.Shorten("https://benchmark.example.com", "550e8400-e29b-41d4-a716-446655440000")

	req := httptest.NewRequest("GET", "/"+shortURL, nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
	}
}

func BenchmarkGetUserURLsHandler(b *testing.B) {
	router, repo := setupBenchRouter(b)
	service := shortener.NewURLService(repo)
	cfg := &config.Config{BaseURL: "http://localhost:8080"}

	router.GET("/api/user/urls", GetUserURLsHandler(service, cfg))

	userID := "550e8400-e29b-41d4-a716-446655440000"
	for i := 0; i < 100; i++ {
		_, _ = service.Shorten("https://example.com/user/"+strconv.Itoa(i), userID)
	}

	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
	}
}

// Полный рабочий вариант — конфликтов больше не будет

func BenchmarkShortenBatch_RealDB(b *testing.B) {
	setupBenchDB(b)
	svc := shortener.NewURLService(benchRepo)
	baseURL := "http://localhost:8080"
	userID := "550e8400-e29b-41d4-a716-446655440000"

	sizes := []struct {
		name string
		n    int
	}{
		{"10", 10}, {"50", 50}, {"100", 100}, {"500", 500}, {"1000", 1000},
	}

	for _, sz := range sizes {
		b.Run(sz.name, func(b *testing.B) {
			b.StopTimer() // не считаем подготовку

			// Один батч на всю жизнь бенчмарка
			reqs := make([]model.URLBatchRequest, sz.n)
			counter := int(time.Now().UnixNano()) // гарантируем уникальность между запусками

			b.StartTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Каждый раз новые уникальные URL
				for j := 0; j < sz.n; j++ {
					reqs[j] = model.URLBatchRequest{
						CorrelationID: strconv.Itoa(j),
						OriginalURL:   "https://bench.example/" + strconv.Itoa(counter),
					}
					counter++
				}

				_, err := shortenBatch(reqs, svc, baseURL, userID)
				if err != nil {
					b.Fatalf("shortenBatch failed: %v", err)
				}
			}
		})
	}
}
