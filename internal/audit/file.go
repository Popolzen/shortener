package audit

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// FileObserver наблюдатель, пишущий в файл
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver создаёт наблюдателя для записи в файл
func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileObserver{file: file}, nil
}

// Notify записывает событие в файл
func (f *FileObserver) Notify(event Event) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("audit file: ошибка сериализации: %v", err)
		return
	}

	data = append(data, '\n')
	if _, err := f.file.Write(data); err != nil {
		log.Printf("audit file: ошибка записи: %v", err)
	}
}

// Close закрывает файл
func (f *FileObserver) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.file.Close()
}
