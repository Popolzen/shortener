package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/config/db"
	"github.com/Popolzen/shortener/internal/handler"
	"github.com/Popolzen/shortener/internal/middleware/compressor"
	"github.com/Popolzen/shortener/internal/middleware/logger"
	"github.com/Popolzen/shortener/internal/repository/filestorage"
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
	dbCfg := db.NewDBConfig(*cfg)

	repo := filestorage.NewURLRepository(cfg.GetFilePath())
	shortener := shortener.NewURLService(repo)

	db, err := db.NewDataBase(*cfg)
	fmt.Print(err)
	err = db.Migrate()
	fmt.Print(err)

	r := gin.Default()

	r.Use(logger.RequestLogger())
	r.Use(compressor.Compresser())
	r.POST("/", handler.PostHandler(shortener, cfg))
	r.POST("/api/shorten", handler.PostHandlerJSON(shortener, cfg))
	r.GET("/:id", handler.GetHandler(shortener))
	r.GET("/ping", handler.PingHandler(dbCfg))

	// Обработка сигналов SIGINT и SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nПолучен сигнал остановки, сохраняем данные...")
		if err := repo.SaveURLToFile(); err != nil {
			log.Printf("Ошибка сохранения при завершении: %v", err)
			os.Exit(1)
		}
		fmt.Println("Данные сохранены, программа завершена.")
		os.Exit(0)
	}()

	addr := cfg.GetAddress()
	log.Printf("URL Shortener запущен на http://%s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

}
