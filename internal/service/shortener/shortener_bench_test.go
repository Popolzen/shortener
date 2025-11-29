package shortener

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/Popolzen/shortener/internal/repository/database"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

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

// BenchmarkShortURL измеряет производительность генерации коротких URL
func BenchmarkShortURL(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = shortURL(6)
	}
}

// BenchmarkShorten измеряет полный цикл сокращения URL
func BenchmarkShorten(b *testing.B) {
	setupBenchDB(b)
	service := NewURLService(benchRepo)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		userID := "550e8400-e29b-41d4-a716-446655440000"
		longURL := "https://example.com/very/long/url/path/" + string(rune(i%1000))
		_, _ = service.Shorten(longURL, userID)
	}
}

// BenchmarkGetLongURL измеряет скорость получения длинного URL
func BenchmarkGetLongURL(b *testing.B) {
	setupBenchDB(b)
	service := NewURLService(benchRepo)

	// Подготовка данных
	userID := "550e8400-e29b-41d4-a716-446655440001"
	shortURL, _ := service.Shorten("https://example.com/gettest", userID)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.GetLongURL(shortURL)
	}
}

// BenchmarkGetFormattedUserURLs измеряет форматирование пользовательских URL
func BenchmarkGetFormattedUserURLs(b *testing.B) {
	setupBenchDB(b)
	service := NewURLService(benchRepo)
	userID := "550e8400-e29b-41d4-a716-446655440002"
	baseURL := "http://localhost:8080"

	// Подготовка: создаем 10 URL для пользователя
	for i := 0; i < 10; i++ {
		_, _ = service.Shorten("https://example.com/formatted/"+string(rune(i)), userID)
	}

	// b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.GetFormattedUserURLs(userID, baseURL)
	}
}

// BenchmarkIsUniq измеряет производительность проверки уникальности
func BenchmarkIsUniq(b *testing.B) {
	setupBenchDB(b)
	service := NewURLService(benchRepo)
	userID := "550e8400-e29b-41d4-a716-446655440003"

	// Заполняем репозиторий данными
	for i := 0; i < 100; i++ {
		_ = benchRepo.Store(shortURL(6), "https://example.com/uniq/"+string(rune(i)), userID)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = service.isUniq(shortURL(6))
	}
}
