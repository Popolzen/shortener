package main

import (
	"fmt"
	"log"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	shortURLs := make(map[string]string)

	r := gin.Default()

	r.POST("/", handler.PostHandler(shortURLs))
	r.GET("/:id", handler.GetHandler(shortURLs))

	cfg := config.NewConfig()

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	log.Printf("URL Shortener запущен на http://%s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

}
