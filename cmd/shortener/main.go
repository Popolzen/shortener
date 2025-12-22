package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	r := setupRouter(shortener, cfg, dbCfg, publisher)

	// HTTP/HTTPS сервер с graceful shutdown
	srv := &http.Server{
		Addr:    cfg.GetAddress(),
		Handler: r,
	}

	// Запуск сервера в горутине
	go func() {
		var err error
		if cfg.EnableHTTPS {
			log.Printf("URL Shortener запущен на https://%s (HTTPS)", cfg.GetAddress())
			err = srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
		} else {
			log.Printf("URL Shortener запущен на http://%s", cfg.GetAddress())
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Graceful Shutdown
	gracefulShutdown(srv, repo, publisher)
}

// gracefulShutdown обрабатывает сигналы завершения
func gracefulShutdown(srv *http.Server, repo repository.URLRepository, pub *audit.Publisher) {
	quit := make(chan os.Signal, 1)
	// Обрабатываем SIGINT, SIGTERM, SIGQUIT
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-quit
	log.Printf("Получен сигнал %v, начинаем graceful shutdown...", sig)

	// Контекст с таймаутом для завершения запросов
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем HTTP сервер (ждём завершения активных запросов)
	log.Println("Останавливаем HTTP сервер...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Ошибка при остановке сервера: %v", err)
	}

	// Закрываем репозиторий
	log.Println("Закрываем репозиторий...")
	closeRepository(repo)

	// Закрываем audit publisher
	log.Println("Закрываем audit publisher...")
	if err := pub.Close(); err != nil {
		log.Printf("Ошибка закрытия publisher: %v", err)
	}

	log.Println("Сервис успешно остановлен")
}

// closeRepository закрывает репозиторий в зависимости от типа
func closeRepository(repo repository.URLRepository) {
	switch r := repo.(type) {
	case *database.URLRepository:
		// Закрываем канал удаления и ждём worker'ов
		r.Shutdown()
		if err := r.DB.Close(); err != nil {
			log.Printf("Ошибка закрытия DB: %v", err)
		}
		log.Println("БД репозиторий закрыт")

	case *filestorage.URLRepository:
		// Сохраняем данные в файл перед выходом
		if err := r.SaveURLToFile(); err != nil {
			log.Printf("Ошибка сохранения в файл: %v", err)
		}
		log.Println("Файловый репозиторий сохранён")

	case *memory.URLRepository:
		// In-memory не требует очистки
		log.Println("In-memory репозиторий не требует сохранения")
	}
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
