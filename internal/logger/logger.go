package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sugar *zap.SugaredLogger

// Init инициализирует zap логгер
func Init() error {
	config := zap.NewProductionConfig()

	// Настройка формата времени
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Настройка уровня логирования
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	logger, err := config.Build()
	if err != nil {
		return err
	}

	sugar = logger.Sugar()
	return nil
}

// RequestResponseLogger — middleware-логер для входящих HTTP-запросов.
func RequestResponseLogger() gin.HandlerFunc {
	return func(c *gin.Context) {

		start := time.Now()
		uri := c.Request.RequestURI
		method := c.Request.Method

		c.Next() // Выполнение следующего handler

		duration := time.Since(start)
		status := c.Writer.Status()
		size := c.Writer.Size()

		sugar.Infoln(
			"uri", uri,
			"method", method,
			"duration", duration,
			"status", status,
			"size", size,
		)

	}
}

func Close() {
	if sugar != nil {
		sugar.Sync()
	}
}
