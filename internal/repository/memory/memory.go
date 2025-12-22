package memory

import (
	"fmt"

	"github.com/Popolzen/shortener/internal/model"
)

type URLRepository struct {
	urls         map[string]string
	correlations map[string]string
}

func (r URLRepository) Get(shortURL string) (string, error) {

	if longURL, exists := r.urls[shortURL]; exists {
		return longURL, nil
	}
	return "", fmt.Errorf("URL not found")
}

func (r *URLRepository) Store(shortURL, longURL, _ string) error {
	r.urls[shortURL] = longURL
	return nil
}

func NewURLRepository() *URLRepository {
	return &URLRepository{
		urls:         map[string]string{},
		correlations: map[string]string{},
	}
}

func (r *URLRepository) StoreBatch() {

}

// memory Repository - заглушки для GetUserURLs
func (r *URLRepository) GetUserURLs(userID string) ([]model.URLPair, error) {
	return nil, fmt.Errorf("GetUserURLs not implemented for in-memory storage")
}

// memory Repository - заглушки для DeleteURLs
func (r *URLRepository) DeleteURLs(userID string, urlIDs []string) {
	fmt.Print("DeteleUrls not implemented for in-memory storage")
}

func (r *URLRepository) Close() error {
	return nil
}
