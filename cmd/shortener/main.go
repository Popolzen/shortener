package main

import (
	"log"

	"github.com/Popolzen/shortener/internal/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	shortURLs := make(map[string]string)

	r := gin.Default()

	r.POST("/", handler.PostHandler(shortURLs))
	r.GET("/:id", handler.GetHandler(shortURLs))

	log.Println("Сервер запущен на порту 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}
}
