package database

import (
	"database/sql"
	"fmt"
)

type urlRepository struct {
	DB *sql.DB
}

func (r urlRepository) Get(shortURL string) (string, error) {

	return "", fmt.Errorf("URL not found")
}

func (r *urlRepository) Store(shortURL, longURL string) error {

	return nil
}

func NewURLRepository(db *sql.DB) *urlRepository {
	return &urlRepository{
		DB: db,
	}
}
