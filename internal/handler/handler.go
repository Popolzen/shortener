package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/service"
	"github.com/gin-gonic/gin"
)

// PostHandler создает короткую ссылку
func PostHandler(shortURLs map[string]string, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Читаем тело запроса
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "Неправильное тело запроса")
			return
		}

		shortURL, err := service.Shortener(string(body), shortURLs)
		if err != nil {
			c.String(http.StatusBadRequest, "Не удалось сгенерить короткую ссылку")
			return
		}

		fullShortURL := cfg.BaseURL + "/" + shortURL
		c.Header("Content-Type", "text/plain")
		c.Header("Content-Length", strconv.Itoa(len(fullShortURL)))
		c.String(http.StatusCreated, fullShortURL)
	}
}

// GetHandler перенаправляет по короткой ссылке
func GetHandler(shortURLs map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {

		shortURL := strings.TrimPrefix(c.Request.URL.Path, "/")

		if longURL, exists := (shortURLs)[shortURL]; exists {
			c.Header("Location", longURL)
			c.Header("Content-Type", "text/plain")
			c.Status(http.StatusTemporaryRedirect)
			return
		}

		c.String(http.StatusBadRequest, "Не нашли ссылку")
	}
}
