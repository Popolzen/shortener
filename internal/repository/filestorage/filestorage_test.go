package filestorage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Popolzen/shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === Helpers ===

func createTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "filestorage_test_*.json")
	require.NoError(t, err)
	if content != "" {
		_, err = f.WriteString(content)
		require.NoError(t, err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "filestorage_test_*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// === NewURLRepository ===

func TestNewURLRepository_EmptyFile(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)

	assert.NotNil(t, repo)
	assert.Empty(t, repo.urls)
	assert.Equal(t, path, repo.path)
}

func TestNewURLRepository_WithExistingData(t *testing.T) {
	data := []model.URLRecord{
		{UUID: "1", ShortURL: "abc", OriginalURL: "https://one.com"},
		{UUID: "2", ShortURL: "def", OriginalURL: "https://two.com"},
	}
	content, _ := json.Marshal(data)
	path := createTempFile(t, string(content))

	repo := NewURLRepository(path)

	assert.Len(t, repo.urls, 2)
	assert.Equal(t, "https://one.com", repo.urls["abc"])
	assert.Equal(t, "https://two.com", repo.urls["def"])
}

func TestNewURLRepository_InvalidJSON(t *testing.T) {
	path := createTempFile(t, "invalid json {{{")

	repo := NewURLRepository(path)

	assert.NotNil(t, repo)
	assert.Empty(t, repo.urls)
}

func TestNewURLRepository_NonexistentFile(t *testing.T) {
	repo := NewURLRepository("/nonexistent/path/file.json")

	assert.NotNil(t, repo)
	assert.Empty(t, repo.urls)
}

func TestNewURLRepository_EmptyArray(t *testing.T) {
	path := createTempFile(t, "[]")

	repo := NewURLRepository(path)

	assert.NotNil(t, repo)
	assert.Empty(t, repo.urls)
}

// === Store ===

func TestStore_Success(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	err := repo.Store("test123", "https://example.com", "user-1")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", repo.urls["test123"])
}

func TestStore_PersistsToFile(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	repo.Store("persisted", "https://persisted.com", "user-1")

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "persisted")
	assert.Contains(t, string(content), "https://persisted.com")
}

func TestStore_MultipleURLs(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	repo.Store("a", "https://a.com", "user-1")
	repo.Store("b", "https://b.com", "user-2")
	repo.Store("c", "https://c.com", "user-1")

	assert.Len(t, repo.urls, 3)

	content, _ := os.ReadFile(path)
	assert.Contains(t, string(content), "https://a.com")
	assert.Contains(t, string(content), "https://b.com")
	assert.Contains(t, string(content), "https://c.com")
}

func TestStore_Overwrite(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	repo.Store("key", "https://old.com", "user")
	repo.Store("key", "https://new.com", "user")

	longURL, _ := repo.Get("key")
	assert.Equal(t, "https://new.com", longURL)
}

func TestStore_CreatesFileIfNotExists(t *testing.T) {
	dir := createTempDir(t)
	path := filepath.Join(dir, "newfile.json")

	repo := NewURLRepository(path)
	err := repo.Store("new", "https://new.com", "user")

	require.NoError(t, err)

	_, statErr := os.Stat(path)
	assert.NoError(t, statErr)
}

// === Get ===

func TestGet_Success(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	repo.urls["found"] = "https://found.com"

	longURL, err := repo.Get("found")

	require.NoError(t, err)
	assert.Equal(t, "https://found.com", longURL)
}

func TestGet_NotFound(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)

	_, err := repo.Get("missing")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGet_AfterStore(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	repo.Store("abc", "https://abc.com", "user")

	longURL, err := repo.Get("abc")

	require.NoError(t, err)
	assert.Equal(t, "https://abc.com", longURL)
}

// === Persistence ===

func TestPersistence_ReloadAfterRestart(t *testing.T) {
	path := createTempFile(t, "")

	// Первый "запуск"
	repo1 := NewURLRepository(path)
	repo1.Store("key1", "https://one.com", "user")
	repo1.Store("key2", "https://two.com", "user")

	// "Перезапуск" — новый репо с тем же файлом
	repo2 := NewURLRepository(path)

	url1, err1 := repo2.Get("key1")
	url2, err2 := repo2.Get("key2")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, "https://one.com", url1)
	assert.Equal(t, "https://two.com", url2)
}

func TestPersistence_OverwritePreserved(t *testing.T) {
	path := createTempFile(t, "")

	repo1 := NewURLRepository(path)
	repo1.Store("key", "https://old.com", "user")
	repo1.Store("key", "https://new.com", "user")

	repo2 := NewURLRepository(path)
	longURL, err := repo2.Get("key")

	require.NoError(t, err)
	assert.Equal(t, "https://new.com", longURL)
}

func TestPersistence_ManyURLs(t *testing.T) {
	path := createTempFile(t, "")

	repo1 := NewURLRepository(path)
	for i := 0; i < 100; i++ {
		key := string(rune('a'+i%26)) + string(rune('0'+i%10))
		repo1.Store(key, "https://example.com/"+key, "user")
	}

	repo2 := NewURLRepository(path)
	assert.Len(t, repo2.urls, 100)
}

// === SaveURLToFile ===

func TestSaveURLToFile_Success(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	repo.urls["x"] = "https://x.com"
	repo.urls["y"] = "https://y.com"

	err := repo.SaveURLToFile()

	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	var records []model.URLRecord
	json.Unmarshal(content, &records)
	assert.Len(t, records, 2)
}

func TestSaveURLToFile_EmptyURLs(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)

	err := repo.SaveURLToFile()

	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Equal(t, "[]", string(content))
}

// === GetUserURLs ===

func TestGetUserURLs_NotImplemented(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)

	urls, err := repo.GetUserURLs("user-1")

	assert.Error(t, err)
	assert.Nil(t, urls)
	assert.Contains(t, err.Error(), "not implemented")
}

// === DeleteURLs ===

func TestDeleteURLs_NotPanics(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)

	assert.NotPanics(t, func() {
		repo.DeleteURLs("user-1", []string{"abc"})
	})
}

func TestDeleteURLs_DoesNotDeleteAnything(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	repo.Store("abc", "https://example.com", "user-1")

	repo.DeleteURLs("user-1", []string{"abc"})

	// В file реализации Delete не работает
	longURL, err := repo.Get("abc")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", longURL)
}

// === Edge cases ===

func TestStore_SpecialCharactersInURL(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	specialURL := "https://example.com/path?q=hello world&foo=bar#section"
	repo.Store("special", specialURL, "user")

	repo2 := NewURLRepository(path)
	got, err := repo2.Get("special")

	require.NoError(t, err)
	assert.Equal(t, specialURL, got)
}

func TestStore_UnicodeInURL(t *testing.T) {
	path := createTempFile(t, "")

	repo := NewURLRepository(path)
	unicodeURL := "https://example.com/путь/到/chemin"
	repo.Store("unicode", unicodeURL, "user")

	repo2 := NewURLRepository(path)
	got, err := repo2.Get("unicode")

	require.NoError(t, err)
	assert.Equal(t, unicodeURL, got)
}
