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
	// cfg.DBurl = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
	// 	`localhost`, 5432, `postgres`, `123456`, `shortener`)

	dbCfg := db.NewDBConfig(*cfg)

	switch {
	case dbCfg.DBurl != "":

		dbInstance, err := db.NewDataBase(*cfg, dbCfg)
		if err != nil {
			log.Fatal("Ошибка подключения к БД:", err)
		}

		if err := dbInstance.Migrate(); err != nil {
			log.Fatal("Ошибка выполнения миграций:", err)
		}

		repo = database.NewURLRepository(dbInstance.DB)

		log.Println("Используется БД репозиторий")
	case cfg.GetFilePath() != "":
		repo = filestorage.NewURLRepository(cfg.GetFilePath())

		log.Println("Используется файл")
	default:
		repo = memory.NewURLRepository()

		log.Println("Используется память")
	}

	shortener := shortener.NewURLService(repo)
	r := gin.Default()

	r.Use(logger.RequestLogger())
	r.Use(compressor.Compresser())
	r.POST("/", handler.PostHandler(shortener, cfg))
	r.POST("/api/shorten", handler.PostHandlerJSON(shortener, cfg))
	r.POST("/api/shorten/batch", handler.BatchHandler(shortener, cfg))
	r.GET("/:id", handler.GetHandler(shortener))
	r.GET("/ping", handler.PingHandler(dbCfg))

	addr := cfg.GetAddress()
	log.Printf("URL Shortener запущен на http://%s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

}
