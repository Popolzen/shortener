// Package handler содержит обработчики HTTP-запросов для сервиса сокращения URL.
//
// Пакет предоставляет функции-обработчики для:
//   - создания коротких ссылок (текстовый и JSON форматы)
//   - получения оригинальных URL по коротким ссылкам
//   - пакетного создания коротких ссылок
//   - получения истории URL пользователя
//   - асинхронного удаления URL
//   - проверки доступности базы данных
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Popolzen/shortener/internal/audit"
	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/db"
	"github.com/Popolzen/shortener/internal/middleware/auth"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/database"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
)

// getUserID извлекает идентификатор пользователя из контекста запроса.
//
// Функция используется в хендлерах для получения userID, установленного
// middleware аутентификации.
//
// Возвращает:
//   - string: идентификатор пользователя
//   - bool: true если userID найден и имеет корректный тип, иначе false
func getUserID(c *gin.Context) (string, bool) {
	val, ok := c.Get(string(auth.UserIDKey))
	if !ok {
		return "", false
	}
	uid, ok := val.(string)
	return uid, ok
}

// PostHandler создает обработчик для сокращения URL в текстовом формате.
//
// Эндпоинт: POST /
// Content-Type: text/plain
//
// Принимает в теле запроса оригинальный URL в виде простого текста
// и возвращает сокращенный URL.
//
// Коды ответа:
//   - 201: URL успешно сокращен, возвращается короткая ссылка
//   - 400: некорректное тело запроса
//   - 409: URL уже существует, возвращается существующая короткая ссылка
//   - 500: внутренняя ошибка сервера
//
// Пример запроса:
//
//	POST / HTTP/1.1
//	Content-Type: text/plain
//
//	https://example.com
//
// Пример ответа:
//
//	HTTP/1.1 201 Created
//	Content-Type: text/plain
//
//	http://localhost:8080/abc123
func PostHandler(urlService shortener.URLService, cfg *config.Config, auditPub *audit.Publisher) gin.HandlerFunc {
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

		longURL := string(body)
		shortURL, err := urlService.Shorten(longURL, userID)

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

		auditPub.Publish(audit.NewEvent(audit.ActionShorten, userID, longURL))
	}

}

// GetHandler создает обработчик для перенаправления по короткой ссылке.
//
// Эндпоинт: GET /{id}
//
// Принимает идентификатор короткой ссылки и перенаправляет на оригинальный URL.
//
// Коды ответа:
//   - 307: перенаправление на оригинальный URL
//   - 404: короткая ссылка не найдена
//   - 410: ссылка была удалена пользователем
//
// Пример запроса:
//
//	GET /abc123 HTTP/1.1
//
// Пример ответа:
//
//	HTTP/1.1 307 Temporary Redirect
//	Location: https://example.com
func GetHandler(urlService shortener.URLService, auditPub *audit.Publisher) gin.HandlerFunc {
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

		userID, _ := getUserID(c)
		auditPub.Publish(audit.NewEvent(audit.ActionFollow, userID, longURL))
	}
}

// GetUserURLsHandler создает обработчик для получения всех URL пользователя.
//
// Эндпоинт: GET /api/user/urls
//
// Возвращает список всех сокращенных URL, созданных текущим пользователем.
// Требуется валидная cookie аутентификации.
//
// Коды ответа:
//   - 200: успешно, возвращается JSON массив с URL
//   - 204: у пользователя нет сохраненных URL
//   - 401: невалидная cookie аутентификации
//   - 500: внутренняя ошибка сервера
//
// Пример ответа:
//
//	HTTP/1.1 200 OK
//	Content-Type: application/json
//
//	[
//	  {
//	    "short_url": "http://localhost:8080/abc123",
//	    "original_url": "https://example.com"
//	  },
//	  {
//	    "short_url": "http://localhost:8080/def456",
//	    "original_url": "https://google.com"
//	  }
//	]
func GetUserURLsHandler(urlService shortener.URLService, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем валидность куки (для 401 статуса)

		hadCookie, _ := c.Get(string(auth.HadCookieKey))
		cookieWasValid, _ := c.Get(string(auth.CookieValidKey))

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

// PostHandlerJSON создает обработчик для сокращения URL в JSON формате.
//
// Эндпоинт: POST /api/shorten
// Content-Type: application/json
//
// Принимает JSON с оригинальным URL и возвращает JSON с короткой ссылкой.
//
// Коды ответа:
//   - 201: URL успешно сокращен
//   - 400: некорректный JSON в теле запроса
//   - 409: URL уже существует
//   - 500: внутренняя ошибка сервера
//
// Пример запроса:
//
//	POST /api/shorten HTTP/1.1
//	Content-Type: application/json
//
//	{
//	  "url": "https://example.com"
//	}
//
// Пример ответа:
//
//	HTTP/1.1 201 Created
//	Content-Type: application/json
//
//	{
//	  "result": "http://localhost:8080/abc123"
//	}
func PostHandlerJSON(urlService shortener.URLService, cfg *config.Config, auditPub *audit.Publisher) gin.HandlerFunc {
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

		auditPub.Publish(audit.NewEvent(audit.ActionShorten, userID, request.URL))
	}

}

// PingHandler создает обработчик для проверки доступности базы данных.
//
// Эндпоинт: GET /ping
//
// Выполняет проверку подключения к базе данных.
//
// Коды ответа:
//   - 200: база данных доступна
//   - 500: база данных недоступна
//
// Пример запроса:
//
//	GET /ping HTTP/1.1
//
// Пример ответа:
//
//	HTTP/1.1 200 OK
func PingHandler(dbconf db.DBConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		err := dbconf.PingDB()
		if err != nil {
			ctx.Status(http.StatusInternalServerError)
		}
		ctx.Status(http.StatusOK)
	}
}

// BatchHandler создает обработчик для пакетного сокращения URL.
//
// Эндпоинт: POST /api/shorten/batch
// Content-Type: application/json
//
// Принимает массив URL для сокращения и возвращает массив результатов.
// Каждый элемент связан через correlation_id.
//
// Коды ответа:
//   - 201: все URL успешно сокращены
//   - 400: некорректный JSON в теле запроса
//   - 500: внутренняя ошибка сервера
//
// Пример запроса:
//
//	POST /api/shorten/batch HTTP/1.1
//	Content-Type: application/json
//
//	[
//	  {
//	    "correlation_id": "1",
//	    "original_url": "https://example.com"
//	  },
//	  {
//	    "correlation_id": "2",
//	    "original_url": "https://google.com"
//	  }
//	]
//
// Пример ответа:
//
//	HTTP/1.1 201 Created
//	Content-Type: application/json
//
//	[
//	  {
//	    "correlation_id": "1",
//	    "short_url": "http://localhost:8080/abc123"
//	  },
//	  {
//	    "correlation_id": "2",
//	    "short_url": "http://localhost:8080/def456"
//	  }
//	]
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

// DeleteURLsHandler создает обработчик для асинхронного удаления URL.
//
// Эндпоинт: DELETE /api/user/urls
// Content-Type: application/json
//
// Принимает массив идентификаторов коротких ссылок для удаления.
// Удаление происходит асинхронно в фоновом режиме.
//
// Коды ответа:
//   - 202: запрос принят, удаление будет выполнено асинхронно
//   - 400: некорректный JSON в теле запроса
//   - 500: внутренняя ошибка сервера
//
// Пример запроса:
//
//	DELETE /api/user/urls HTTP/1.1
//	Content-Type: application/json
//
//	["abc123", "def456", "ghi789"]
//
// Пример ответа:
//
//	HTTP/1.1 202 Accepted
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

// shortenBatch выполняет пакетное сокращение URL.
//
// Принимает массив запросов и возвращает массив ответов,
// где каждый элемент связан через correlation_id.
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

// handleConflictError обрабатывает ошибку конфликта URL.
//
// Проверяет, является ли ошибка конфликтом (URL уже существует),
// и возвращает полный URL существующей короткой ссылки.
//
// Возвращает:
//   - string: полный URL существующей короткой ссылки
//   - bool: true если это конфликт, иначе false
func handleConflictError(err error, baseURL string) (string, bool) {
	var conflictErr database.ErrURLConflictError
	if errors.As(err, &conflictErr) {
		return baseURL + "/" + conflictErr.ExistingShortURL, true
	}
	return "", false
}
