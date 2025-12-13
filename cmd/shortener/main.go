package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"net/http"
	_ "net/http/pprof"

	"github.com/Popolzen/shortener/internal/audit"
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

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()

	// Инициализируем логгер
	if err := logger.Init(); err != nil {
		log.Fatal("Не удалось инициализировать логгер:", err)
	}
	defer logger.Close()

	gin.SetMode(gin.ReleaseMode)
	cfg := config.NewConfig()
	dbCfg := db.NewDBConfig(*cfg)

	// Запускаем pprof сервер на настраиваемом порту
	if cfg.PprofAddr != "" {
		go func() {
			log.Printf("pprof сервер запущен на http://%s/debug/pprof/", cfg.PprofAddr)
			if err := http.ListenAndServe(cfg.PprofAddr, nil); err != nil {
				log.Printf("Ошибка запуска pprof сервера: %v", err)
			}
		}()
	}

	// Инициализируем паблишера
	publisher := initAudit(cfg)

	// Инициализируем репозиторий
	repo := initRepository(cfg, dbCfg)

	// Создаем сервис
	shortener := shortener.NewURLService(repo)

	// Настраиваем роутер
	r := setupRouter(shortener, cfg, dbCfg, publisher)

	// Запускаем сервер
	addr := cfg.GetAddress()
	log.Printf("URL Shortener запущен на http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Не удалось запустить сервер:", err)
	}

	// Graceful Shutdown
	setupGracefulShutdown(repo)
}

func printBuildInfo() {
	version := "N/A"
	date := "N/A"
	commit := "N/A"

	if buildVersion != "" {
		version = buildVersion
	}
	if buildDate != "" {
		date = buildDate
	}
	if buildCommit != "" {
		commit = buildCommit
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
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

// initAudit - функция инициализации аудита:
func initAudit(cfg *config.Config) *audit.Publisher {
	publisher := audit.NewPublisher()

	// Файловый observer
	if cfg.GetAuditFile() != "" {
		fileObs, err := audit.NewFileObserver(cfg.GetAuditFile())
		if err != nil {
			log.Printf("Не удалось создать file observer: %v", err)
		} else {
			publisher.Subscribe(fileObs)
			log.Printf("Аудит в файл: %s", cfg.GetAuditFile())
		}
	}

	// HTTP observer
	if cfg.GetAuditURL() != "" {
		httpObs := audit.NewHTTPObserver(cfg.GetAuditURL())
		publisher.Subscribe(httpObs)
		log.Printf("Аудит на сервер: %s", cfg.GetAuditURL())
	}

	return publisher
}

// setupRouter настраивает роуты и middleware
func setupRouter(shortener shortener.URLService, cfg *config.Config, dbCfg db.DBConfig, auditPub *audit.Publisher) *gin.Engine {
	r := gin.Default()
	r.Use(logger.RequestLogger())
	r.Use(compressor.Compresser())
	r.Use(auth.AuthMiddleware(cfg))

	r.POST("/", handler.PostHandler(shortener, cfg, auditPub))
	r.POST("/api/shorten", handler.PostHandlerJSON(shortener, cfg, auditPub))
	r.POST("/api/shorten/batch", handler.BatchHandler(shortener, cfg))
	r.GET("/:id", handler.GetHandler(shortener, auditPub))
	r.GET("/api/user/urls", handler.GetUserURLsHandler(shortener, cfg))
	r.DELETE("/api/user/urls", handler.DeleteURLsHandler(shortener))

	r.GET("/ping", handler.PingHandler(dbCfg))
	return r
}
