package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewURLRepository(t *testing.T) {
	repo := NewURLRepository()

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.urls)
	assert.Empty(t, repo.urls)
}

func TestStore_AndGet(t *testing.T) {
	repo := NewURLRepository()

	err := repo.Store("abc123", "https://example.com", "user-1")
	require.NoError(t, err)

	longURL, err := repo.Get("abc123")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", longURL)
}

func TestStore_MultipleURLs(t *testing.T) {
	repo := NewURLRepository()

	repo.Store("a", "https://one.com", "user-1")
	repo.Store("b", "https://two.com", "user-2")
	repo.Store("c", "https://three.com", "user-1")

	url1, err1 := repo.Get("a")
	url2, err2 := repo.Get("b")
	url3, err3 := repo.Get("c")

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	assert.Equal(t, "https://one.com", url1)
	assert.Equal(t, "https://two.com", url2)
	assert.Equal(t, "https://three.com", url3)
}

func TestStore_Overwrite(t *testing.T) {
	repo := NewURLRepository()

	repo.Store("key", "https://old.com", "user-1")
	repo.Store("key", "https://new.com", "user-1")

	longURL, _ := repo.Get("key")
	assert.Equal(t, "https://new.com", longURL)
}

func TestGet_NotFound(t *testing.T) {
	repo := NewURLRepository()

	_, err := repo.Get("notexists")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetUserURLs_NotImplemented(t *testing.T) {
	repo := NewURLRepository()

	urls, err := repo.GetUserURLs("user-1")

	assert.Error(t, err)
	assert.Nil(t, urls)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestDeleteURLs_NotPanics(t *testing.T) {
	repo := NewURLRepository()

	// Не должно паниковать
	assert.NotPanics(t, func() {
		repo.DeleteURLs("user-1", []string{"abc", "def"})
	})
}

func TestStore_IgnoresUserID(t *testing.T) {
	repo := NewURLRepository()

	// userID игнорируется в memory реализации
	repo.Store("x", "https://x.com", "user-1")
	repo.Store("y", "https://y.com", "user-2")

	// Оба URL доступны без привязки к пользователю
	url1, _ := repo.Get("x")
	url2, _ := repo.Get("y")

	assert.Equal(t, "https://x.com", url1)
	assert.Equal(t, "https://y.com", url2)
}
