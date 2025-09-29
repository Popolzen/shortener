package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/db"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/database"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
)

// getUserID извлекает userID
func getUserID(c *gin.Context) (string, bool) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		return "", false
	}

	return userID, true
}

// PostHandler создает короткую ссылку
func PostHandler(urlService shortener.URLService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Читаем тело запроса
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusBadRequest, "Неправильное тело запроса")
			return
		}

		userID, ok := getUserID(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		shortURL, err := urlService.Shorten(string(body), userID)

		if fullShortURL, isConflict := handleConflictError(err, cfg.BaseURL); isConflict {
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
			if errors.Is(err, model.ErrURLDeleted) { // ссылка deleted
				c.Status(http.StatusGone) // 410
				return
			}
			c.String(http.StatusNotFound, "Не нашли ссылку")
			return
		}

		c.Header("Location", longURL)
		c.Header("Content-Type", "text/plain")
		c.Status(http.StatusTemporaryRedirect)
	}
}

// GetUserURLsHandler возвращает все URL пользователя
func GetUserURLsHandler(urlService shortener.URLService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем валидность куки (для 401 статуса)
		hadCookie, _ := c.Get("had_cookie")
		cookieWasValid, _ := c.Get("cookie_was_valid")

		// Если была кука, но она невалидная - 401
		if hadCookie.(bool) && !cookieWasValid.(bool) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Получаем userID из контекста
		userID, ok := getUserID(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Получаем отформатированные URL через сервис
		urls, err := urlService.GetFormattedUserURLs(userID, cfg.BaseURL)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		// Если URL нет - возвращаем 204 No Content
		if len(urls) == 0 {
			c.Status(http.StatusNoContent)
			return
		}

		// Возвращаем список URL
		c.JSON(http.StatusOK, urls)
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

		userID, ok := getUserID(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		shortURL, err := urlService.Shorten(request.URL, userID)

		// Проверяем, является ли ошибка конфликтом URL
		if fullShortURL, isConflict := handleConflictError(err, cfg.BaseURL); isConflict {
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

		userID, ok := getUserID(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		responseBatch, err := shortenBatch(requestBatch, urlService, cfg.GetBaseURL(), userID)

		if err != nil {
			c.String(http.StatusBadRequest, "Не удалось сгенерить короткую ссылку")
			return
		}

		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusCreated, responseBatch)
		c.Header("Content-Length", strconv.Itoa(len(responseBatch)))

	}
}

func DeleteURLsHandler(urlService shortener.URLService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := getUserID(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		var shortURLs []string
		if err := json.NewDecoder(c.Request.Body).Decode(&shortURLs); err != nil {
			c.String(http.StatusBadRequest, "Неправильное тело запроса")
			return
		}

		// Вызываем метод repository для асинхронного удаления
		urlService.DeleteURLsAsync(userID, shortURLs)

		c.Status(http.StatusAccepted)
	}
}

// shortenBatch сокращает батч ссылок
func shortenBatch(req []model.URLBatchRequest, urlService shortener.URLService, baseURL string, userID string) ([]model.URLBatchResponse, error) {
	response := make([]model.URLBatchResponse, 0, len(req))
	for _, request := range req {
		shortURL, err := urlService.Shorten(request.OriginalURL, userID)
		if err != nil {
			return nil, err
		}
		response = append(response, model.URLBatchResponse{CorrelationID: request.CorrelationID, ShortURL: baseURL + "/" + shortURL})
	}
	return response, nil
}

// handleConflictError обрабатывает ошибку конфликта URL
func handleConflictError(err error, baseURL string) (string, bool) {
	var conflictErr database.ErrURLConflictError
	if errors.As(err, &conflictErr) {
		return baseURL + "/" + conflictErr.ExistingShortURL, true
	}
	return "", false
}
