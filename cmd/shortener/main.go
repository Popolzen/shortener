package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/db"
	"github.com/Popolzen/shortener/internal/handler"
	"github.com/Popolzen/shortener/internal/middleware/auth"
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
	// Инициализируем логгер
	if err := logger.Init(); err != nil {
		log.Fatal("Не удалось инициализировать логгер:", err)
	}
	defer logger.Close()

	gin.SetMode(gin.ReleaseMode)
	cfg := config.NewConfig()
	dbCfg := db.NewDBConfig(*cfg)
	// Инициализируем репозиторий
	repo := initRepository(cfg, dbCfg)

	// Создаем сервис
	shortener := shortener.NewURLService(repo)

	// Настраиваем роутер
	r := setupRouter(shortener, cfg, dbCfg)

	// Запускаем сервер
	addr := cfg.GetAddress()
	log.Printf("URL Shortener запущен на http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

	// Graceful Shutdown
	setupGracefulShutdown(repo)
}

// initRepository инициализирует репозиторий в зависимости от конфигурации
func initRepository(cfg *config.Config, dbCfg db.DBConfig) repository.URLRepository {
	var repo repository.URLRepository

	// dbCfg.DBurl = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
	// 	`localhost`, 5432, `postgres`, `123456`, `shortener`)

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

	return repo
}

// GracefulShutdown
func setupGracefulShutdown(repo repository.URLRepository) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Получен сигнал остановки, завершаем работу...")
		if dbRepo, ok := repo.(*database.URLRepository); ok {
			dbRepo.Shutdown() // Закрываем канал и ждём worker'ов
			if err := dbRepo.DB.Close(); err != nil {
				log.Printf("Ошибка закрытия DB: %v", err)
			}
		}
		log.Println("Сервис остановлен gracefully")
		os.Exit(0)
	}()
}

// setupRouter настраивает роуты и middleware
func setupRouter(shortener shortener.URLService, cfg *config.Config, dbCfg db.DBConfig) *gin.Engine {
	r := gin.Default()
	r.Use(logger.RequestLogger())
	r.Use(compressor.Compresser())
	r.Use(auth.AuthMiddleware())

	r.POST("/", handler.PostHandler(shortener, cfg))
	r.POST("/api/shorten", handler.PostHandlerJSON(shortener, cfg))
	r.POST("/api/shorten/batch", handler.BatchHandler(shortener, cfg))
	r.GET("/:id", handler.GetHandler(shortener))
	r.GET("/api/user/urls", handler.GetUserURLsHandler(shortener, cfg))
	r.DELETE("/api/user/urls", handler.DeleteURLsHandler(shortener))

	r.GET("/ping", handler.PingHandler(dbCfg))
	return r
}
