package memory

import "fmt"

type URLRepository struct {
	urls         map[string]string
	correlations map[string]string
}

func (r URLRepository) Get(shortURL, _ string) (string, error) {

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
