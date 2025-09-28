package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Popolzen/shortener/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
)

type ErrURLConflictError struct {
	ExistingShortURL string
}

func (e ErrURLConflictError) Error() string {
	return fmt.Sprintf("URL уже существует с коротким URL: %s", e.ExistingShortURL)
}

type URLRepository struct {
	DB            *sql.DB
	DeleteChannel chan model.DeleteTask
}

// Get получает длинный URL по короткому
func (r *URLRepository) Get(shortURL string) (string, error) {
	var longURL string
	query := `SELECT long_url FROM shortened_urls WHERE short_url = $1 `

	err := r.DB.QueryRow(query, shortURL).Scan(&longURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("ошибка при получении URL: %w", err)
	}

	return longURL, nil
}

// getByLongURL получает короткий URL по длинному
func (r *URLRepository) getByLongURL(longURL string) (string, error) {
	var shortURL string
	query := `SELECT short_url FROM shortened_urls WHERE long_url = $1`
	err := r.DB.QueryRow(query, longURL).Scan(&shortURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("URL not found")
		}
		return "", fmt.Errorf("ошибка при получении короткого URL: %w", err)
	}
	return shortURL, nil
}

// Store сохраняет соответствие короткого и длинного URL
func (r *URLRepository) Store(shortURL, longURL, id string) error {
	query := `
    INSERT INTO shortened_urls (short_url, long_url, created_at, user_id)
    VALUES ($1, $2, $3, $4)
`

	now := time.Now()
	_, err := r.DB.Exec(query, shortURL, longURL, now, id)
	if err != nil {

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			existingShortURL, getErr := r.getByLongURL(longURL)
			if getErr != nil {
				return fmt.Errorf("ошибка при получении существующего URL: %w", getErr)
			}
			return ErrURLConflictError{ExistingShortURL: existingShortURL}
		}

		return fmt.Errorf("ошибка при сохранении URL: %w", err)
	}

	return nil
}

// GetUserURLs - возвращает все URLs для конкретного пользователя
func (r *URLRepository) GetUserURLs(userID string) ([]model.URLPair, error) {
	query := `SELECT short_url, long_url FROM shortened_urls WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.DB.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении URL пользователя: %w", err)
	}
	// defer rows.Close()

	var urls []model.URLPair
	for rows.Next() {
		var pair model.URLPair
		err := rows.Scan(&pair.ShortURL, &pair.OriginalURL)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("URL not found")
			}
			return nil, fmt.Errorf("ошибка при получении короткого URL: %w", err)
		}
		urls = append(urls, pair)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по строкам: %w", err)
	}

	return urls, nil
}

func NewURLRepository(db *sql.DB) *URLRepository {
	return &URLRepository{
		DB: db,
	}
}

func (r *URLRepository) InitDeleteSystem() {
	r.DeleteChannel = make(chan model.DeleteTask, 1000) // Буфер на 1000
	// Запускаем несколько воркеров
	for i := range 1 {
		go r.deleteWorker()
		log.Printf("Worker %d поднялся и готов к работе!", i)
	}

}

func (r *URLRepository) deleteWorker() {
	const batchSize = 100
	const batchTimeout = 2 * time.Second

	taskBuffer := make([]model.DeleteTask, 0, batchSize)
	timer := time.NewTimer(batchTimeout)
	defer timer.Stop()

	for {
		select {
		case task, ok := <-r.DeleteChannel:
			if !ok {
				return
			} // Если канал закрыт, выходим
			taskBuffer = append(taskBuffer, task)
			if len(taskBuffer) >= batchSize {
				r.processBatch(taskBuffer)
				taskBuffer = taskBuffer[:0]
			}
			timer.Reset(batchTimeout) // Reset после добавления

		case <-timer.C: // Тикаем 2 секунды, и записываем неполный батч, если не набралось
			if len(taskBuffer) > 0 {
				r.processBatch(taskBuffer)
				taskBuffer = taskBuffer[:0]
			}
			timer.Reset(batchTimeout) // Reset для следующего
		}
	}
}
func (r *URLRepository) processBatch(tasks []model.DeleteTask) {
	if len(tasks) == 0 {
		return
	}
	// Группируем по userID
	groups := make(map[string][]string)
	for _, task := range tasks {
		groups[task.UserID] = append(groups[task.UserID], task.ShortURL)
	}
	// Для каждой группы
	for userID, shortURLs := range groups {
		err := r.batchDeleteURLs(userID, shortURLs)
		if err != nil {
			log.Printf("Ошибка в батче для user %s: %v", userID, err)
		}
	}
}

func (r *URLRepository) batchDeleteURLs(userID string, shortURLs []string) error {
	if len(shortURLs) == 0 {
		return nil
	}

	query := `
        UPDATE shortened_urls 
        SET is_deleted = true 
        WHERE user_id = $1 AND short_url = ANY($2) AND is_deleted = false
    `

	_, err := r.DB.Exec(query, userID, pq.Array(shortURLs))
	return err
}

// Асинхронное удаление - отправка в канал
func (r *URLRepository) DeteleUrls(userID string, urlIDs []string) {
	for _, shortURL := range urlIDs {
		select {
		case r.DeleteChannel <- model.DeleteTask{UserID: userID, ShortURL: shortURL}:
		default:
			log.Printf("Delete channel full, task dropped: %s", shortURL)
		}
	}
}
