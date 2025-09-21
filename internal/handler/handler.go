package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/config/db"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/database"
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

		var conflictErr database.URLConflictError
		if errors.As(err, &conflictErr) {
			fullShortURL := cfg.BaseURL + "/" + conflictErr.ExistingShortURL
			c.Header("Content-Type", "text/plain")
			c.Header("Content-Length", strconv.Itoa(len(fullShortURL)))
			c.String(http.StatusConflict, fullShortURL)
			return
		}
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

		// Проверяем, является ли ошибка конфликтом URL
		var conflictErr database.URLConflictError
		if errors.As(err, &conflictErr) {
			fullShortURL := cfg.BaseURL + "/" + conflictErr.ExistingShortURL
			response := model.Result{
				Result: fullShortURL,
			}
			c.Header("Content-Type", "application/json")
			c.JSON(http.StatusConflict, response)
			return
		}

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

// PingHandler - хэндлер пинга.
func PingHandler(dbconf db.DBConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		err := dbconf.PingDB()
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
		}
		ctx.Status(http.StatusOK)
	}
}

// BatchHandler - хэндрер батчей
func BatchHandler(urlService shortener.URLService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {

		var requestBatch []model.URLBatchRequest
		var responseBatch []model.URLBatchResponse

		if err := json.NewDecoder(c.Request.Body).Decode(&requestBatch); err != nil {
			c.String(http.StatusBadRequest, "Неправильное тело запроса")
			return
		}

		// Красивый вывод для дебага
		if debugJSON, err := json.MarshalIndent(requestBatch, "", "  "); err == nil {
			fmt.Printf("DEBUG RequestBatch:\n%s\n", string(debugJSON))
		}

		responseBatch, err := shortenBatch(requestBatch, urlService, cfg.GetBaseURL())

		if err != nil {
			c.String(http.StatusBadRequest, "Не удалось сгенерить короткую ссылку")
			return
		}

		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusCreated, responseBatch)
		c.Header("Content-Length", strconv.Itoa(len(responseBatch)))

	}
}

func shortenBatch(req []model.URLBatchRequest, urlService shortener.URLService, baseUrl string) ([]model.URLBatchResponse, error) {
	response := make([]model.URLBatchResponse, 0, len(req))
	for _, request := range req {
		shortURL, err := urlService.Shorten(request.OriginalURL)
		if err != nil {
			return nil, err
		}
		response = append(response, model.URLBatchResponse{CorrelationID: request.CorrelationID, ShortURL: baseUrl + "/" + shortURL})
	}
	return response, nil
}
