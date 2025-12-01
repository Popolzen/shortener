package shortener

import (
	"testing"

	"github.com/Popolzen/shortener/internal/repository/memory"
)

// BenchmarkShortenInMemory — полный цикл сокращения
func BenchmarkShortenInMemory(b *testing.B) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	userID := "test-user-123"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		longURL := "https://example.com/path/" + string(rune(i%10000))
		_, _ = service.Shorten(longURL, userID)
	}
}

// BenchmarkGetLongURLInMemory — получение длинного URL
func BenchmarkGetLongURLInMemory(b *testing.B) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	userID := "test-user-123"

	shortURL, _ := service.Shorten("https://example.com/test", userID)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.GetLongURL(shortURL)
	}
}

// BenchmarkIsUniqInMemory — проверка уникальности
func BenchmarkIsUniqInMemory(b *testing.B) {
	repo := memory.NewURLRepository()
	service := NewURLService(repo)
	userID := "test-user-123"

	// Заполняем репозиторий
	for i := 0; i < 100; i++ {
		_ = repo.Store(shortURL(6), "https://example.com/"+string(rune(i)), userID)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = service.isUniq(shortURL(6))
	}
}
