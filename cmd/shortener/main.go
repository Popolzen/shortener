package main

import (
	"log"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/config/db"
	"github.com/Popolzen/shortener/internal/handler"
	"github.com/Popolzen/shortener/internal/middleware/compressor"
	"github.com/Popolzen/shortener/internal/middleware/logger"
	"github.com/Popolzen/shortener/internal/repository"
	"github.com/Popolzen/shortener/internal/repository/database"
	"github.com/Popolzen/shortener/internal/repository/filestorage"
	"github.com/Popolzen/shortener/internal/repository/memory"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
)

func main() {
	var repo repository.URLRepository
	// Инициализируем логгер
	if err := logger.Init(); err != nil {
		log.Fatal("Не удалось инициализировать логгер:", err)
	}
	defer logger.Close()

	gin.SetMode(gin.ReleaseMode)

	cfg := config.NewConfig()
	dbCfg := db.NewDBConfig(*cfg)

	switch {
	case dbCfg.DBurl != "":

		dbInstance, err := db.NewDataBase(*cfg)
		if err != nil {
			log.Fatal("Ошибка подключения к БД:", err)
		}

		if err := dbInstance.Migrate(); err != nil {
			log.Printf("Ошибка выполнения миграций:", err)
		}

		repo = database.NewURLRepository(dbInstance.DB)

	case cfg.GetFilePath() != "":
		repo = filestorage.NewURLRepository(cfg.GetFilePath())
	default:
		repo = memory.NewURLRepository()
	}
	// repo := filestorage.NewURLRepository(cfg.GetFilePath())
	shortener := shortener.NewURLService(repo)
	r := gin.Default()

	r.Use(logger.RequestLogger())
	r.Use(compressor.Compresser())
	r.POST("/", handler.PostHandler(shortener, cfg))
	r.POST("/api/shorten", handler.PostHandlerJSON(shortener, cfg))
	r.GET("/:id", handler.GetHandler(shortener))
	r.GET("/ping", handler.PingHandler(dbCfg))

	addr := cfg.GetAddress()
	log.Printf("URL Shortener запущен на http://%s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

}
