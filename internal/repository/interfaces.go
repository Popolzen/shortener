package repository

import "github.com/Popolzen/shortener/internal/model"

//go:generate mockgen -destination=mocks/mock_repository.go -package=mocks github.com/Popolzen/shortener/internal/repository URLRepository

type URLRepository interface {
	Store(shortURL, longURL, userID string) error
	Get(shortURL string) (string, error)
	GetUserURLs(userID string) ([]model.URLPair, error)
	DeleteURLs(userID string, urlIDs []string)
}
