package shortener

import (
	"errors"
	"testing"

	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// === Тесты с gomock (взаимодействие с репозиторием) ===

func TestShorten_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found"))
	repo.EXPECT().Store(gomock.Any(), "https://example.com", "user-123").Return(nil)

	service := NewURLService(repo)
	shortURL, err := service.Shorten("https://example.com", "user-123")

	require.NoError(t, err)
	assert.Len(t, shortURL, 6)
}

func TestShorten_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found"))
	repo.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db error"))

	service := NewURLService(repo)
	_, err := service.Shorten("https://example.com", "user-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestShorten_RetryOnCollision(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	// Первые 2 раза URL существует, третий — свободен
	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any()).Return("exists", nil),
		repo.EXPECT().Get(gomock.Any()).Return("exists", nil),
		repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found")),
	)

	repo.EXPECT().Store(gomock.Any(), "https://example.com", "user-1").Return(nil)

	service := NewURLService(repo)
	shortURL, err := service.Shorten("https://example.com", "user-1")

	require.NoError(t, err)
	assert.Len(t, shortURL, 6)
}

func TestGetLongURL_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().Get("abc123").Return("https://example.com", nil)

	service := NewURLService(repo)
	longURL, err := service.GetLongURL("abc123")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", longURL)
}

func TestGetLongURL_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().Get("missing").Return("", errors.New("not found"))

	service := NewURLService(repo)
	_, err := service.GetLongURL("missing")

	assert.Error(t, err)
}

func TestGetLongURL_Deleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().Get("deleted").Return("", model.ErrURLDeleted)

	service := NewURLService(repo)
	_, err := service.GetLongURL("deleted")

	assert.ErrorIs(t, err, model.ErrURLDeleted)
}

func TestGetFormattedUserURLs_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().GetUserURLs("user-1").Return([]model.URLPair{
		{ShortURL: "abc", OriginalURL: "https://one.com"},
		{ShortURL: "def", OriginalURL: "https://two.com"},
	}, nil)

	service := NewURLService(repo)
	urls, err := service.GetFormattedUserURLs("user-1", "http://localhost:8080")

	require.NoError(t, err)
	require.Len(t, urls, 2)
	assert.Equal(t, "http://localhost:8080/abc", urls[0].ShortURL)
	assert.Equal(t, "http://localhost:8080/def", urls[1].ShortURL)
	// OriginalURL не меняется
	assert.Equal(t, "https://one.com", urls[0].OriginalURL)
}

func TestGetFormattedUserURLs_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().GetUserURLs("unknown").Return(nil, nil)

	service := NewURLService(repo)
	urls, err := service.GetFormattedUserURLs("unknown", "http://localhost")

	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestGetFormattedUserURLs_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().GetUserURLs("user-1").Return(nil, errors.New("db error"))

	service := NewURLService(repo)
	_, err := service.GetFormattedUserURLs("user-1", "http://localhost")

	assert.Error(t, err)
}

func TestDeleteURLsAsync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().DeleteURLs("user-123", []string{"a", "b", "c"})

	service := NewURLService(repo)
	service.DeleteURLsAsync("user-123", []string{"a", "b", "c"})
}

func TestDeleteURLsAsync_EmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)
	repo.EXPECT().DeleteURLs("user-123", []string{})

	service := NewURLService(repo)
	service.DeleteURLsAsync("user-123", []string{})
}

// === Тесты без моков (чистая логика генератора) ===

func TestShortURL_Length(t *testing.T) {
	tests := []int{4, 6, 8, 10, 20}

	for _, length := range tests {
		result := shortURL(length)
		assert.Len(t, result, length)
	}
}

func TestShortURL_ValidCharset(t *testing.T) {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	for i := 0; i < 100; i++ {
		result := shortURL(6)
		for _, c := range result {
			assert.Contains(t, charset, string(c))
		}
	}
}

func TestShortURL_Randomness(t *testing.T) {
	results := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		results[shortURL(6)] = true
	}
	// При 62^6 комбинациях 1000 должны быть почти все уникальны
	assert.Greater(t, len(results), 990)
}

func TestShortURL_ZeroLength(t *testing.T) {
	result := shortURL(0)
	assert.Empty(t, result)
}
