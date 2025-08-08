package main

import (
	"log"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	shortURLs := make(map[string]string)
	gin.SetMode(gin.ReleaseMode)

	cfg := config.NewConfig()

	r := gin.Default()
	r.POST("/", handler.PostHandler(shortURLs, cfg))
	r.GET("/:id", handler.GetHandler(shortURLs))

	addr := cfg.Address()
	log.Printf("URL Shortener запущен на http://%s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

}
