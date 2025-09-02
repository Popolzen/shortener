package handler

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
)

// PostHandler создает короткую ссылку
func PostHandler(urlService shortener.URLService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Читаем тело запроса
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "Неправильное тело запроса")
			return
		}

		shortURL, err := urlService.Shorten(string(body))
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
func GetHandler(urlService shortener.URLService) gin.HandlerFunc {
	return func(c *gin.Context) {

		shortURL := strings.TrimPrefix(c.Request.URL.Path, "/")

		longURL, err := urlService.GetLongURL(shortURL)
		if err != nil {
			c.String(http.StatusNotFound, "Не нашли ссылку")
			return
		}

		c.Header("Location", longURL)
		c.Header("Content-Type", "text/plain")
		c.Status(http.StatusTemporaryRedirect)
	}
}

// PostHandlerJSON создает короткую ссылку, принимает json, возвращает json.
func PostHandlerJSON(urlService shortener.URLService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request model.URL

		if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
			c.String(http.StatusBadRequest, "Неправильное тело запроса")
			return
		}
		shortURL, err := urlService.Shorten(request.URL)
		if err != nil {
			c.String(http.StatusBadRequest, "Не удалось сгенерить короткую ссылку")
			return
		}

		fullShortURL := cfg.BaseURL + "/" + shortURL

		response := model.Result{
			Result: fullShortURL,
		}
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusCreated, response)
		c.Header("Content-Length", strconv.Itoa(len(fullShortURL)))
	}

}

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	contentType := g.Header().Get("Content-Type")

	if strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html") {
		g.Header().Set("Content-Encoding", "gzip")
		return g.writer.Write(data)
	}
	return g.ResponseWriter.Write(data)
}
func (g *gzipWriter) Close() error {
	return g.writer.Close()
}

func CompressHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		fmt.Printf("Accept-Encoding: '%s'\n", acceptEncoding)
		fmt.Printf("Empty string contains gzip: %v\n", strings.Contains("", "gzip"))
		fmt.Printf("AcceptEncoding != '': %v\n", acceptEncoding != "")
		fmt.Printf("Should compress: %v\n", strings.Contains(strings.ToLower(acceptEncoding), "gzip") && acceptEncoding != "")

		c.Next()
	}
}
