package database

import (
	"database/sql"
	"fmt"
	"time"
)

type urlRepository struct {
	DB *sql.DB
}

// Get получает длинный URL по короткому
func (r *urlRepository) Get(shortURL string) (string, error) {
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

// Store сохраняет соответствие короткого и длинного URL
func (r *urlRepository) Store(shortURL, longURL string) error {
	query := `
    INSERT INTO shortened_urls (short_url, long_url, created_at)
    VALUES ($1, $2, $3)
    ON CONFLICT (short_url)
    DO UPDATE SET
        long_url = EXCLUDED.long_url
`

	now := time.Now()
	_, err := r.DB.Exec(query, shortURL, longURL, now)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении URL: %w", err)
	}

	return nil
}

func NewURLRepository(db *sql.DB) *urlRepository {
	return &urlRepository{
		DB: db,
	}
}
