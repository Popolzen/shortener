package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type ErrURLConflictError struct {
	ExistingShortURL string
}

func (e ErrURLConflictError) Error() string {
	return fmt.Sprintf("URL уже существует с коротким URL: %s", e.ExistingShortURL)
}

type URLRepository struct {
	DB *sql.DB
}

// Get получает длинный URL по короткому
func (r *URLRepository) Get(shortURL string) (string, error) {
	var longURL string
	query := `SELECT long_url FROM shortened_urls WHERE short_url = $1`

	err := r.DB.QueryRow(query, shortURL).Scan(&longURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("ошибка при получении URL: %w", err)
	}

	return longURL, nil
}

// getByLongURL получает короткий URL по длинному
func (r *URLRepository) getByLongURL(longURL string) (string, error) {
	var shortURL string
	query := `SELECT short_url FROM shortened_urls WHERE long_url = $1`
	err := r.DB.QueryRow(query, longURL).Scan(&shortURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("ошибка при получении короткого URL: %w", err)
	}
	return shortURL, nil
}

// Store сохраняет соответствие короткого и длинного URL
func (r *URLRepository) Store(shortURL, longURL string) error {
	query := `
    INSERT INTO shortened_urls (short_url, long_url, created_at)
    VALUES ($1, $2, $3)

`

	now := time.Now()
	_, err := r.DB.Exec(query, shortURL, longURL, now)
	if err != nil {

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			existingShortURL, getErr := r.getByLongURL(longURL)
			if getErr != nil {
				return fmt.Errorf("ошибка при получении существующего URL: %w", getErr)
			}
			return ErrURLConflictError{ExistingShortURL: existingShortURL}
		}

		return fmt.Errorf("ошибка при сохранении URL: %w", err)
	}

	return nil
}

func NewURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{
		DB: db,
	}
}
