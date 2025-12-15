package model

import "errors"

type URL struct {
	URL string `json:"url"`
}

type Result struct {
	Result string `json:"result"`
}

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// generate:reset
type URLBatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type URLBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// URLPair представляет пару сокращённого и оригинального URL
type URLPair struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// DeleteTask стурктура таски для удаления
type DeleteTask struct {
	UserID   string
	ShortURL string
}

// Простая кастомная ошибка
var ErrURLDeleted = errors.New("URL has been deleted")
