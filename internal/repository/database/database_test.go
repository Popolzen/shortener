package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Popolzen/shortener/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// === Setup ===

// setupTestDB поднимает PostgreSQL в Docker и возвращает подключение.
// Контейнер автоматически остановится после теста.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	// 1. Запускаем контейнер PostgreSQL
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine", // образ
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			// Ждём пока БД будет готова принимать подключения
			// "database system is ready" появляется дважды в логах postgres
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// 2. Регистрируем остановку контейнера после теста
	t.Cleanup(func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	})

	// 3. Получаем строку подключения
	// Формат: postgres://test:test@localhost:55432/testdb
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 4. Подключаемся к БД
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	// 5. Создаём схему (как в твоей миграции)
	createSchema(t, db)

	return db
}

// createSchema создаёт таблицу как в миграции
func createSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shortened_urls (
			id BIGSERIAL PRIMARY KEY,
			user_id UUID NOT NULL,
			long_url TEXT UNIQUE NOT NULL,
			short_url VARCHAR(20) UNIQUE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			is_deleted BOOL DEFAULT FALSE,
			
			CONSTRAINT chk_short_url_length CHECK (length(short_url) >= 4)
		);
		
		CREATE UNIQUE INDEX IF NOT EXISTS idx_shortened_urls_short_url 
			ON shortened_urls(short_url);
		CREATE INDEX IF NOT EXISTS idx_shortened_urls_user_id 
			ON shortened_urls(user_id);
	`)
	require.NoError(t, err)
}

// cleanupTable очищает таблицу между тестами
func cleanupTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("TRUNCATE shortened_urls RESTART IDENTITY")
	require.NoError(t, err)
}

// createTestRepo создаёт репозиторий для тестов
func createTestRepo(t *testing.T, db *sql.DB) *URLRepository {
	t.Helper()
	repo := &URLRepository{
		DB:            db,
		DeleteChannel: make(chan model.DeleteTask, 100),
	}
	// Не запускаем воркеры для простоты тестов
	return repo
}

// === Store ===

func TestStore_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)

	err := repo.Store("abcd12", "https://example.com", "550e8400-e29b-41d4-a716-446655440000")

	require.NoError(t, err)

	// Проверяем что записалось в БД
	var longURL string
	err = db.QueryRow("SELECT long_url FROM shortened_urls WHERE short_url = $1", "abcd12").Scan(&longURL)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", longURL)
}

func TestStore_MultipleURLs(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	repo.Store("aaaa11", "https://one.com", userID)
	repo.Store("bbbb22", "https://two.com", userID)
	repo.Store("cccc33", "https://three.com", userID)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM shortened_urls").Scan(&count)
	assert.Equal(t, 3, count)
}

func TestStore_DuplicateShortURL_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	err1 := repo.Store("dupl12", "https://first.com", userID)
	require.NoError(t, err1)

	err2 := repo.Store("dupl12", "https://second.com", userID)
	assert.Error(t, err2)
}

func TestStore_DuplicateLongURL_ReturnsConflict(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	// Убираем UNIQUE constraint с long_url для этого теста,
	// или добавляем его в схему. Смотри свою миграцию.
	// В твоей миграции long_url UNIQUE, поэтому:

	err1 := repo.Store("first1", "https://duplicate.com", userID)
	require.NoError(t, err1)

	err2 := repo.Store("second", "https://duplicate.com", userID)

	var conflictErr ErrURLConflictError
	assert.ErrorAs(t, err2, &conflictErr)
	assert.Equal(t, "first1", conflictErr.ExistingShortURL)
}

func TestStore_ShortURLTooShort_Error(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	// Constraint: length(short_url) >= 4
	err := repo.Store("abc", "https://example.com", userID)

	assert.Error(t, err)
}

// === Get ===

func TestGet_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	repo.Store("test12", "https://example.com", userID)

	longURL, err := repo.Get("test12")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", longURL)
}

func TestGet_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)

	_, err := repo.Get("notfound")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGet_DeletedURL_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	repo.Store("delt12", "https://example.com", userID)

	// Помечаем как удалённый
	_, err := db.Exec("UPDATE shortened_urls SET is_deleted = true WHERE short_url = $1", "delt12")
	require.NoError(t, err)

	_, err = repo.Get("delt12")

	assert.ErrorIs(t, err, model.ErrURLDeleted)
}

// === GetUserURLs ===

func TestGetUserURLs_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	repo.Store("usr111", "https://one.com", userID)
	repo.Store("usr222", "https://two.com", userID)

	urls, err := repo.GetUserURLs(userID)

	require.NoError(t, err)
	assert.Len(t, urls, 2)
}

func TestGetUserURLs_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)

	urls, err := repo.GetUserURLs("550e8400-e29b-41d4-a716-446655440000")

	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestGetUserURLs_OnlyOwnURLs(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	user1 := "550e8400-e29b-41d4-a716-446655440001"
	user2 := "550e8400-e29b-41d4-a716-446655440002"

	repo.Store("u1url1", "https://user1-one.com", user1)
	repo.Store("u1url2", "https://user1-two.com", user1)
	repo.Store("u2url1", "https://user2-one.com", user2)

	urls, err := repo.GetUserURLs(user1)

	require.NoError(t, err)
	assert.Len(t, urls, 2)

	for _, u := range urls {
		assert.Contains(t, u.OriginalURL, "user1")
	}
}

func TestGetUserURLs_OrderByCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	repo.Store("old111", "https://old.com", userID)
	time.Sleep(10 * time.Millisecond) // небольшая задержка
	repo.Store("new111", "https://new.com", userID)

	urls, err := repo.GetUserURLs(userID)

	require.NoError(t, err)
	require.Len(t, urls, 2)
	// ORDER BY created_at DESC — новые первые
	assert.Equal(t, "new111", urls[0].ShortURL)
	assert.Equal(t, "old111", urls[1].ShortURL)
}

// === DeleteURLs ===

func TestDeleteURLs_SendsToChannel(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	repo.DeleteURLs(userID, []string{"abc123", "def456"})

	assert.Len(t, repo.DeleteChannel, 2)

	task1 := <-repo.DeleteChannel
	assert.Equal(t, userID, task1.UserID)
	assert.Equal(t, "abc123", task1.ShortURL)

	task2 := <-repo.DeleteChannel
	assert.Equal(t, "def456", task2.ShortURL)
}

func TestDeleteURLs_EmptySlice(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)

	repo.DeleteURLs("user", []string{})

	assert.Empty(t, repo.DeleteChannel)
}

// === batchDeleteURLs ===

func TestBatchDeleteURLs_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	repo.Store("del111", "https://one.com", userID)
	repo.Store("del222", "https://two.com", userID)
	repo.Store("keep11", "https://keep.com", userID)

	err := repo.batchDeleteURLs(userID, []string{"del111", "del222"})

	require.NoError(t, err)

	// Проверяем что удалённые помечены
	_, err1 := repo.Get("del111")
	_, err2 := repo.Get("del222")
	url3, err3 := repo.Get("keep11")

	assert.ErrorIs(t, err1, model.ErrURLDeleted)
	assert.ErrorIs(t, err2, model.ErrURLDeleted)
	require.NoError(t, err3)
	assert.Equal(t, "https://keep.com", url3)
}

func TestBatchDeleteURLs_OnlyOwnURLs(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	user1 := "550e8400-e29b-41d4-a716-446655440001"
	user2 := "550e8400-e29b-41d4-a716-446655440002"

	repo.Store("u1only", "https://user1.com", user1)
	repo.Store("u2only", "https://user2.com", user2)

	// user2 пытается удалить URL user1
	err := repo.batchDeleteURLs(user2, []string{"u1only"})

	require.NoError(t, err) // Ошибки нет, просто ничего не удалилось

	// URL user1 не удалён
	url, err := repo.Get("u1only")
	require.NoError(t, err)
	assert.Equal(t, "https://user1.com", url)
}

func TestBatchDeleteURLs_EmptySlice(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)

	err := repo.batchDeleteURLs("user", []string{})

	require.NoError(t, err)
}

// === Edge cases ===

func TestStore_SpecialCharactersInURL(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	specialURL := "https://example.com/path?q=hello%20world&foo=bar#section"
	err := repo.Store("spec12", specialURL, userID)
	require.NoError(t, err)

	got, err := repo.Get("spec12")
	require.NoError(t, err)
	assert.Equal(t, specialURL, got)
}

func TestStore_UnicodeInURL(t *testing.T) {
	db := setupTestDB(t)
	repo := createTestRepo(t, db)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	unicodeURL := "https://example.com/путь/到/chemin"
	err := repo.Store("unic12", unicodeURL, userID)
	require.NoError(t, err)

	got, err := repo.Get("unic12")
	require.NoError(t, err)
	assert.Equal(t, unicodeURL, got)
}
