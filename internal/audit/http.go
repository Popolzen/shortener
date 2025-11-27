package audit

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// HTTPObserver наблюдатель, отправляющий на удалённый сервер
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создаёт наблюдателя для отправки на HTTP endpoint
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify отправляет событие на удалённый сервер
func (h *HTTPObserver) Notify(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit http: ошибка сериализации: %v", err)
		return
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("audit http: ошибка отправки: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("audit http: сервер вернул %d", resp.StatusCode)
	}
}

// Close для HTTP ничего не делает
func (h *HTTPObserver) Close() error {
	return nil
}
