package filestorage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Popolzen/shortener/internal/model"
	"github.com/google/uuid"
)

type urlRepository struct {
	urls map[string]string
	path string
}

func (r urlRepository) Get(shortURL string) (string, error) {

	if longURL, exists := r.urls[shortURL]; exists {
		return longURL, nil
	}
	return "", fmt.Errorf("URL not found")
}

func (r *urlRepository) Store(shortURL, longURL string) error {
	var urlRecord model.URLRecord

	r.urls[shortURL] = longURL

	urlRecord.UUID = uuid.New().String()
	urlRecord.ShortURL = shortURL
	urlRecord.OriginalURL = longURL

	r.saveURLToFile(r.path, urlRecord)
	return nil
}

func NewURLRepository(path string) *urlRepository {
	var repo urlRepository

	repo.setPath(path)
	repo.urls = map[string]string{}

	err := repo.loadURLs(path)

	if err != nil {
		return &urlRepository{
			urls: map[string]string{},
			path: path,
		}
	}
	return &repo
}

// loadURLs - загружает данные из файла в память.
func (r *urlRepository) loadURLs(path string) error {
	var urlRecord []model.URLRecord

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644) // создаем файл если его нет
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("ошибка инфо файла: %w", err)
	}

	if fileInfo.Size() == 0 { // если файл пустой - инициализируем []
		if _, err := file.WriteString("[]"); err != nil {
			return fmt.Errorf("failed to initialize file: %w", err)
		}
	}

	if _, err := file.Seek(0, 0); err != nil { // Переводим каретку в начало если файл был пустой и записали
		return fmt.Errorf("ошибка перемещения указателя файла: %w", err)
	}

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

// saveURLToFile  запись по url в файл
func (r *urlRepository) saveURLToFile(path string, url model.URLRecord) error {
	data, err := json.Marshal(url)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644) // создаем файл если его нет
	if err != nil {
		return fmt.Errorf("ошибка открытия файла: %w", err)
	}
	defer file.Close()

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("ошибка инфо файла: %w", err)
	}

	if fileInfo.Size() == 2 { // если файл содержит только []
		if _, err := file.Seek(1, 0); err != nil {
			return fmt.Errorf("ошибка перемещения указателя файла: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("ошибка записи данных в файл: %w", err)
		}
		if _, err := file.WriteString("]"); err != nil {
			return fmt.Errorf("ошибка записи закрывающей скобки: %w", err)
		}
		return nil
	}

	if _, err := file.Seek(fileInfo.Size()-1, 0); err != nil {
		return fmt.Errorf("ошибка перемещения указателя файла: %w", err)
	}
	if _, err := file.WriteString(","); err != nil {
		return fmt.Errorf("ошибка записи запятой: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("ошибка записи данных в файл: %w", err)
	}
	if _, err := file.WriteString("]"); err != nil {
		return fmt.Errorf("ошибка записи закрывающей скобки: %w", err)
	}

	return nil
}

// Устанавливаем путь
func (r *urlRepository) setPath(path string) {
	r.path = path
}
