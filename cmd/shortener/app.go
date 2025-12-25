package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/repository"
)

type App struct {
	server    *http.Server
	repo      repository.URLRepository
	publisher *audit.Publisher
}

// Close закрывает все ресурсы
func (a *App) Close() error {
	log.Println("Закрываем репозиторий...")
	if err := a.repo.Close(); err != nil {
		log.Printf("Ошибка закрытия репозитория: %v", err)
	}

	log.Println("Закрываем audit publisher...")
	if err := a.publisher.Close(); err != nil {
		log.Printf("Ошибка закрытия publisher: %v", err)
	}

	return nil
}

// Shutdown выполняет graceful shutdown с таймаутом
func (a *App) Shutdown(ctx context.Context) error {
	log.Println("Останавливаем HTTP сервер...")
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("ошибка остановки сервера: %w", err)
	}
	return a.Close()
}
