package main

import (
	"log"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/handler"
	"github.com/Popolzen/shortener/internal/logger"
	"github.com/Popolzen/shortener/internal/repository/memory"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
)

func main() {

	// Инициализируем логгер
	if err := logger.Init(); err != nil {
		log.Fatal("Не удалось инициализировать логгер:", err)
	}
	defer logger.Close()

	gin.SetMode(gin.ReleaseMode)

	cfg := config.NewConfig()
	repo := memory.NewURLRepository()
	shortener := shortener.NewURLService(repo)

	r := gin.Default()

	r.Use(logger.RequestResponseLogger())
	r.POST("/", handler.PostHandler(shortener, cfg))
	r.GET("/:id", handler.GetHandler(shortener))

	addr := cfg.Address()
	log.Printf("URL Shortener запущен на http://%s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

}
