package memory

import "fmt"

type urlRepository struct {
	urls         map[string]string
	correlations map[string]string
}

func (r urlRepository) Get(shortURL, _ string) (string, error) {

	if longURL, exists := r.urls[shortURL]; exists {
		return longURL, nil
	}
	return "", fmt.Errorf("URL not found")
}

func (r *urlRepository) Store(shortURL, longURL, _ string) error {
	r.urls[shortURL] = longURL
	return nil
}

func NewURLRepository() *urlRepository {
	return &urlRepository{
		urls:         map[string]string{},
		correlations: map[string]string{},
	}
}

func (r *urlRepository) StoreBatch() {

}
