package filestorage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Popolzen/shortener/internal/model"
	"github.com/google/uuid"
)

type URLRepository struct {
	urls map[string]string
	path string
}

func (r URLRepository) Get(shortURL string) (string, error) {

	if longURL, exists := r.urls[shortURL]; exists {
		return longURL, nil
	}
	return "", fmt.Errorf("URL not found")
}

func (r *URLRepository) Store(shortURL, longURL, _ string) error {

	r.urls[shortURL] = longURL
	r.SaveURLToFile()
	return nil
}

func NewURLRepository(path string) *URLRepository {
	var repo URLRepository

	repo.path = path
	repo.urls = map[string]string{}

	err := repo.loadURLs(path)

	if err != nil {
		return &URLRepository{
			urls: map[string]string{},
			path: path,
		}
	}
	return &repo
}

// loadURLs - загружает данные из файла в память.
func (r *URLRepository) loadURLs(path string) error {
	var urlRecord []model.URLRecord

	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	if err := json.Unmarshal(data, &urlRecord); err != nil {
		return fmt.Errorf("ошибка десериализации JSON: %w", err)
	}
	for i := range urlRecord {
		r.urls[urlRecord[i].ShortURL] = urlRecord[i].OriginalURL
	}

	return nil
}

// SaveURLToFile  запись по url в файл
func (r *URLRepository) SaveURLToFile() error {
	urls := make([]model.URLRecord, 0, len(r.urls))

	for key, value := range r.urls {
		urls = append(urls, model.URLRecord{UUID: uuid.New().String(), OriginalURL: value, ShortURL: key})
	}

	data, err := json.Marshal(urls)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	file, err := os.OpenFile(r.path, os.O_RDWR|os.O_CREATE, 0644) // создаем файл если его нет
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	file.Write(data)

	return nil
}

// FileStorage Repository - заглушки для GetUserURLs
func (r *URLRepository) GetUserURLs(userID string) ([]model.URLPair, error) {
	return nil, fmt.Errorf("GetUserURLs not implemented for file storage")
}

// FileStorage Repository - заглушки для DeleteURLs
func (r *URLRepository) DeleteURLs(userID string, urlIDs []string) {
	fmt.Print("DeteleUrls not implemented for in-memory storage")
}

func (r *URLRepository) Close() error {
	return r.SaveURLToFile() // Сохраняем данные перед закрытием
}
