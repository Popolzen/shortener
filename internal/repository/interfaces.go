package repository

import "github.com/Popolzen/shortener/internal/model"

type URLRepository interface {
	Store(shortURL, longURL, userID string) error
	Get(shortURL string) (string, error)
	GetUserURLs(userID string) ([]model.URLPair, error)
}
