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

func TestShorten_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	// Get вернёт ошибку = URL уникален
	repo.EXPECT().
		Get(gomock.Any()).
		Return("", errors.New("not found"))

	// Store должен быть вызван с правильными аргументами
	repo.EXPECT().
		Store(gomock.Len(6), "https://example.com", "user-123").
		Return(nil)

	service := NewURLService(repo)
	shortURL, err := service.Shorten("https://example.com", "user-123")

	require.NoError(t, err)
	assert.Len(t, shortURL, 6)
}

func TestShorten_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		Get(gomock.Any()).
		Return("", errors.New("not found"))

	repo.EXPECT().
		Store(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("db connection failed"))

	service := NewURLService(repo)
	_, err := service.Shorten("https://example.com", "user-123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestShorten_RetryOnCollision(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	// Первые 2 раза URL существует, третий — уникален
	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any()).Return("https://exists.com", nil),
		repo.EXPECT().Get(gomock.Any()).Return("https://exists.com", nil),
		repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found")),
	)

	repo.EXPECT().
		Store(gomock.Any(), "https://example.com", "user-1").
		Return(nil)

	service := NewURLService(repo)
	shortURL, err := service.Shorten("https://example.com", "user-1")

	require.NoError(t, err)
	assert.Len(t, shortURL, 6)
}

func TestGetLongURL_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		Get("abc123").
		Return("https://example.com", nil)

	service := NewURLService(repo)
	longURL, err := service.GetLongURL("abc123")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", longURL)
}

func TestGetLongURL_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		Get("notexists").
		Return("", errors.New("URL not found"))

	service := NewURLService(repo)
	_, err := service.GetLongURL("notexists")

	assert.Error(t, err)
}

func TestGetLongURL_Deleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		Get("deleted123").
		Return("", model.ErrURLDeleted)

	service := NewURLService(repo)
	_, err := service.GetLongURL("deleted123")

	assert.ErrorIs(t, err, model.ErrURLDeleted)
}

func TestGetFormattedUserURLs_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		GetUserURLs("user-123").
		Return([]model.URLPair{
			{ShortURL: "abc", OriginalURL: "https://one.com"},
			{ShortURL: "def", OriginalURL: "https://two.com"},
		}, nil)

	service := NewURLService(repo)
	urls, err := service.GetFormattedUserURLs("user-123", "http://localhost:8080")

	require.NoError(t, err)
	require.Len(t, urls, 2)
	assert.Equal(t, "http://localhost:8080/abc", urls[0].ShortURL)
	assert.Equal(t, "http://localhost:8080/def", urls[1].ShortURL)
	assert.Equal(t, "https://one.com", urls[0].OriginalURL)
}

func TestGetFormattedUserURLs_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		GetUserURLs("unknown").
		Return([]model.URLPair{}, nil)

	service := NewURLService(repo)
	urls, err := service.GetFormattedUserURLs("unknown", "http://localhost")

	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestGetFormattedUserURLs_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		GetUserURLs("user-1").
		Return(nil, errors.New("db error"))

	service := NewURLService(repo)
	_, err := service.GetFormattedUserURLs("user-1", "http://localhost")

	assert.Error(t, err)
}

func TestDeleteURLsAsync(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		DeleteURLs("user-123", []string{"a", "b", "c"})

	service := NewURLService(repo)
	service.DeleteURLsAsync("user-123", []string{"a", "b", "c"})
}

func TestIsUniq_True(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		Get("newurl").
		Return("", errors.New("not found"))

	service := NewURLService(repo)
	assert.True(t, service.isUniq("newurl"))
}

func TestIsUniq_False(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockURLRepository(ctrl)

	repo.EXPECT().
		Get("existing").
		Return("https://exists.com", nil)

	service := NewURLService(repo)
	assert.False(t, service.isUniq("existing"))
}

// === Тесты shortURL генератора (без моков) ===

func TestShortURL_Length(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 4", 4},
		{"length 6", 6},
		{"length 8", 8},
		{"length 10", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shortURL(tt.length)
			assert.Len(t, result, tt.length)
		})
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
	// 1000 генераций должны дать почти все уникальные
	assert.Greater(t, len(results), 990)
}
