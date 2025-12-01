// Package repository определяет интерфейсы для работы с хранилищем данных.
//
// Пакет следует паттерну Repository и предоставляет абстракцию над различными
// источниками данных: in-memory хранилище, файловое хранилище, базы данных.
package repository

import "github.com/Popolzen/shortener/internal/model"

// URLRepository определяет интерфейс для работы с хранилищем URL.
//
// Интерфейс предоставляет методы для:
//   - сохранения новых URL
//   - получения URL по идентификатору
//   - получения всех URL пользователя
//   - удаления URL
//
// Реализации:
//   - memory.URLRepository: in-memory хранилище
//   - filestorage.URLRepository: файловое хранилище в JSON
//   - database.URLRepository: PostgreSQL хранилище
//
// Пример использования:
//
//	var repo repository.URLRepository
//	repo = memory.NewURLRepository()
//	err := repo.Store("abc123", "https://example.com", "user123")
type URLRepository interface {
	// Store сохраняет связь между короткой и длинной ссылкой.
	//
	// Параметры:
	//   - shortURL: идентификатор короткой ссылки
	//   - longURL: оригинальный URL
	//   - userID: идентификатор пользователя-владельца
	//
	// Возвращает:
	//   - error: ошибку при сохранении или database.ErrURLConflictError если URL уже существует
	//
	// Пример:
	//   err := repo.Store("abc123", "https://example.com", "user123")
	Store(shortURL, longURL, userID string) error

	// Get возвращает оригинальный URL по короткой ссылке.
	//
	// Параметры:
	//   - shortURL: идентификатор короткой ссылки
	//
	// Возвращает:
	//   - string: оригинальный URL
	//   - error: ошибку если ссылка не найдена или model.ErrURLDeleted если ссылка удалена
	//
	// Пример:
	//   longURL, err := repo.Get("abc123")
	//   if errors.Is(err, model.ErrURLDeleted) {
	//       // Обработка удаленной ссылки
	//   }
	Get(shortURL string) (string, error)

	// GetUserURLs возвращает все URL пользователя.
	//
	// Параметры:
	//   - userID: идентификатор пользователя
	//
	// Возвращает:
	//   - []model.URLPair: массив пар коротких и оригинальных URL
	//   - error: ошибку при получении данных
	//
	// Примечание: для in-memory и файлового хранилища возвращает ошибку "not implemented"
	//
	// Пример:
	//   urls, err := repo.GetUserURLs("user123")
	GetUserURLs(userID string) ([]model.URLPair, error)

	// DeleteURLs выполняет удаление URL (для БД - асинхронно).
	//
	// Параметры:
	//   - userID: идентификатор пользователя-владельца
	//   - urlIDs: массив идентификаторов коротких ссылок для удаления
	//
	// Примечание:
	//   - Для database.URLRepository удаление происходит асинхронно через систему воркеров
	//   - Для memory и filestorage реализации это заглушка
	//
	// Пример:
	//   repo.DeleteURLs("user123", []string{"abc123", "def456"})
	DeleteURLs(userID string, urlIDs []string)
}
