package handler

import (
	"compress/gzip"
	"encoding/json"
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
	writer     *gzip.Writer
	compressed bool
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	contentType := g.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
		if !g.compressed {
			g.Header().Set("Content-Encoding", "gzip")
			g.compressed = true
		}
		return g.writer.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipWriter) Close() error {
	if g.compressed {
		return g.writer.Close()
	}
	return nil
}

// CompressHandler обрабатывает gzip сжатие
func CompressHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Распаковка входящего запроса
		if strings.Contains(strings.ToLower(c.Request.Header.Get("Content-Encoding")), "gzip") {
			newReader, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.String(http.StatusBadRequest, "Не удалось распаковать данные")
				return
			}
			c.Request.Body = newReader
			defer newReader.Close()
		}

		// 2. Подготовка сжатия ответа
		acceptEncoding := c.Request.Header.Get("Accept-Encoding")
		if strings.Contains(strings.ToLower(acceptEncoding), "gzip") && acceptEncoding != "" {
			gzipResp := &gzipWriter{
				ResponseWriter: c.Writer,
				writer:         gzip.NewWriter(c.Writer),
				compressed:     false,
			}
			c.Writer = gzipResp
			defer gzipResp.Close()
		}

		c.Next()
	}
}
