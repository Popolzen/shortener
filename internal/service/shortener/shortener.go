package shortener

import (
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type URLService struct {
	repo repository.URLRepository
}

func NewURLService(repo repository.URLRepository) URLService {
	return URLService{repo: repo}
}

// isUniq проверяет что ссылки уже нет
func (s URLService) isUniq(shortURL string) bool {
	_, err := s.repo.Get(shortURL)
	return err != nil
}

// Функция которая делает ссылку короткой и сохраняет ее в мапу
func (s URLService) Shorten(longURL string, id string) (string, error) {
	const length = 6
	const maxAttempts = 1000

	for range maxAttempts {
		su := shortURL(length)
		if s.isUniq(su) {
			err := s.repo.Store(su, longURL, id)
			if err != nil {
				return "", err
			}
			return su, nil
		}
	}

	return "", fmt.Errorf("не удалось создать уникальную ссылку за %d попыток", maxAttempts)
}

func (s URLService) GetFormattedUserURLs(userID string, baseURL string) ([]model.URLPair, error) {
	urls, err := s.GetUserURLs(userID)
	if err != nil {
		return nil, err
	}
	for i := range urls {
		fullShortURL := baseURL + "/" + urls[i].ShortURL
		urls[i].ShortURL = fullShortURL
	}
	return urls, nil
}

func (s URLService) GetLongURL(shortURL string) (string, error) {
	value, err := s.repo.Get(shortURL)
	return value, err
}

// GetUserURLs возвращает все URL конкретного пользователя
func (s *URLService) GetUserURLs(userID string) ([]model.URLPair, error) {
	return s.repo.GetUserURLs(userID)
}

func (s *URLService) DeleteURLsAsync(userID string, shortURLs []string) {
	s.repo.DeleteURLs(userID, shortURLs)
}

var shortURLPool = sync.Pool{
	New: func() any {
		return new([32]byte)
	},
}

func shortURL(length int) string {
	arr := shortURLPool.Get().(*[32]byte)

	for i := 0; i < length; i++ {
		arr[i] = charset[rand.IntN(62)]
	}

	s := string(arr[:length])

	shortURLPool.Put(arr)
	return s
}
