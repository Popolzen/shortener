package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Popolzen/shortener/internal/model"
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
	query := `SELECT long_url FROM shortened_urls WHERE short_url = $1 `

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
func (r *URLRepository) Store(shortURL, longURL, id string) error {
	query := `
    INSERT INTO shortened_urls (short_url, long_url, created_at, user_id)
    VALUES ($1, $2, $3, $4)

`

	now := time.Now()
	_, err := r.DB.Exec(query, shortURL, longURL, now, id)
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

// GetUserURLs - возвращает все URLs для конкретного пользователя
func (r *URLRepository) GetUserURLs(userID string) ([]model.URLPair, error) {
	query := `SELECT short_url, long_url FROM shortened_urls WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении URL пользователя: %w", err)
	}
	// defer rows.Close()

	var urls []model.URLPair
	for rows.Next() {
		var pair model.URLPair
		err := rows.Scan(&pair.ShortURL, &pair.OriginalURL)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("URL not found")
			}
			return nil, fmt.Errorf("ошибка при получении короткого URL: %w", err)
		}
		urls = append(urls, pair)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return urls, nil
}

func NewURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{
		DB: db,
	}
}
