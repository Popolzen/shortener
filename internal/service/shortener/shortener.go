// Package shortener содержит бизнес-логику сервиса сокращения URL.
//
// Пакет предоставляет URLService, который отвечает за:
//   - генерацию уникальных коротких ссылок
//   - сохранение связей между короткими и оригинальными URL
//   - получение оригинальных URL по коротким ссылкам
//   - управление URL пользователей
//   - асинхронное удаление URL
package shortener

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"

	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// URLService предоставляет методы для работы с сокращенными URL.
//
// Сервис является слоем бизнес-логики между обработчиками HTTP-запросов
// и репозиторием хранения данных.
type URLService struct {
	repo repository.URLRepository
}

// NewURLService создает новый экземпляр URLService.
//
// Параметры:
//   - repo: репозиторий для хранения URL
//
// Возвращает:
//   - URLService: новый экземпляр сервиса
//
// Пример использования:
//
//	repo := memory.NewURLRepository()
//	service := shortener.NewURLService(repo)
func NewURLService(repo repository.URLRepository) URLService {
	return URLService{repo: repo}
}

// isUniq проверяет уникальность короткой ссылки.
//
// Возвращает true, если короткая ссылка еще не используется.
func (s URLService) isUniq(shortURL string) bool {
	_, err := s.repo.Get(shortURL)
	return err != nil
}

// Shorten создает короткую ссылку для заданного URL.
//
// Метод генерирует уникальный идентификатор длиной 6 символов,
// проверяет его уникальность и сохраняет связь в репозитории.
// При коллизии выполняется до 1000 попыток генерации.
//
// Параметры:
//   - longURL: оригинальный URL для сокращения
//   - id: идентификатор пользователя
//
// Возвращает:
//   - string: короткий идентификатор URL (без базового адреса)
//   - error: ошибка при генерации или сохранении
//
// Пример использования:
//
//	shortURL, err := service.Shorten("https://example.com", "user123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Короткая ссылка:", shortURL) // Выведет что-то вроде: "abc123"
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

// GetFormattedUserURLs возвращает отформатированные URL пользователя.
//
// Метод получает все URL пользователя из репозитория и добавляет
// базовый URL к каждой короткой ссылке.
//
// Параметры:
//   - userID: идентификатор пользователя
//   - baseURL: базовый URL сервиса (например, "http://localhost:8080")
//
// Возвращает:
//   - []model.URLPair: массив пар коротких и оригинальных URL
//   - error: ошибка при получении данных
//
// Пример использования:
//
//	urls, err := service.GetFormattedUserURLs("user123", "http://localhost:8080")
//	for _, url := range urls {
//	    fmt.Printf("%s -> %s\n", url.ShortURL, url.OriginalURL)
//	}
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

// GetLongURL возвращает оригинальный URL по короткой ссылке.
//
// Параметры:
//   - shortURL: идентификатор короткой ссылки
//
// Возвращает:
//   - string: оригинальный URL
//   - error: ошибка если ссылка не найдена или была удалена
//
// Пример использования:
//
//	longURL, err := service.GetLongURL("abc123")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Оригинальный URL:", longURL)
func (s URLService) GetLongURL(shortURL string) (string, error) {
	value, err := s.repo.Get(shortURL)
	return value, err
}

// GetUserURLs возвращает все URL конкретного пользователя.
//
// Параметры:
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - []model.URLPair: массив пар коротких и оригинальных URL
//   - error: ошибка при получении данных
func (s *URLService) GetUserURLs(userID string) ([]model.URLPair, error) {
	return s.repo.GetUserURLs(userID)
}

// DeleteURLsAsync выполняет асинхронное удаление URL.
//
// Метод помещает задачи на удаление в очередь и немедленно возвращает управление.
// Фактическое удаление выполняется фоновыми воркерами.
//
// Параметры:
//   - userID: идентификатор пользователя
//   - shortURLs: массив идентификаторов коротких ссылок для удаления
//
// Пример использования:
//
//	service.DeleteURLsAsync("user123", []string{"abc123", "def456"})
//	// Метод вернется немедленно, удаление произойдет в фоне
func (s *URLService) DeleteURLsAsync(userID string, shortURLs []string) {
	s.repo.DeleteURLs(userID, shortURLs)
}

var builderPool = sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// shortURL генерирует случайный идентификатор заданной длины.
//
// Использует алфавитно-цифровые символы (a-z, A-Z, 0-9) для генерации.
//
// Параметры:
//   - length: длина генерируемого идентификатора
//
// Возвращает:
//   - string: случайный идентификатор
func shortURL(length int) string {
	// Берём Builder из пула
	b := builderPool.Get().(*strings.Builder)
	defer func() {
		b.Reset()
		builderPool.Put(b)
	}()

	// Выделяем память заранее
	b.Grow(length)

	// Заполняем
	for i := 0; i < length; i++ {
		b.WriteByte(charset[rand.IntN(62)])
	}

	return b.String()
}

// GetStats возвращает статистику сервиса.
//
// Возвращает:
//   - urls: количество сокращенных URL в сервисе
//   - users: количество пользователей в сервисе
//   - error: ошибку при получении статистики
//
// Пример использования:
//
//	urls, users, err := service.GetStats()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("URLs: %d, Users: %d\n", urls, users)
func (s *URLService) GetStats() (urls int, users int, err error) {
	return s.repo.GetStats()
}
