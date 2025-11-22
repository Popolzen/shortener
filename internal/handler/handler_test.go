package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/Popolzen/shortener/internal/repository/mocks"
	"github.com/Popolzen/shortener/internal/service/shortener"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupTestRouter(ctrl *gomock.Controller) (*gin.Engine, *mocks.MockURLRepository) {
	gin.SetMode(gin.TestMode)
	repo := mocks.NewMockURLRepository(ctrl)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", "test-user-123")
		c.Set("had_cookie", true)
		c.Set("cookie_was_valid", true)
		c.Next()
	})

	return router, repo
}

func testConfig() *config.Config {
	return &config.Config{BaseURL: "http://localhost:8080"}
}

// === GetHandler ===

func TestGetHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)
	repo.EXPECT().Get("abc123").Return("https://example.com", nil)

	urlService := shortener.NewURLService(repo)
	router.GET("/:id", GetHandler(urlService))

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))
}

func TestGetHandler_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)
	repo.EXPECT().Get("notfound").Return("", errors.New("not found"))

	urlService := shortener.NewURLService(repo)
	router.GET("/:id", GetHandler(urlService))

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetHandler_Deleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)
	repo.EXPECT().Get("deleted").Return("", model.ErrURLDeleted)

	urlService := shortener.NewURLService(repo)
	router.GET("/:id", GetHandler(urlService))

	req := httptest.NewRequest(http.MethodGet, "/deleted", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusGone, w.Code)
}

// === PostHandler ===

func TestPostHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)

	repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found"))
	repo.EXPECT().Store(gomock.Any(), "https://example.com", "test-user-123").Return(nil)

	urlService := shortener.NewURLService(repo)
	router.POST("/", PostHandler(urlService, testConfig()))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, strings.HasPrefix(w.Body.String(), "http://localhost:8080/"))
}

func TestPostHandler_StoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)

	repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found"))
	repo.EXPECT().Store(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db error"))

	urlService := shortener.NewURLService(repo)
	router.POST("/", PostHandler(urlService, testConfig()))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === PostHandlerJSON ===

func TestPostHandlerJSON_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)

	repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found"))
	repo.EXPECT().Store(gomock.Any(), "https://example.com", "test-user-123").Return(nil)

	urlService := shortener.NewURLService(repo)
	router.POST("/api/shorten", PostHandlerJSON(urlService, testConfig()))

	body, _ := json.Marshal(model.URL{URL: "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response model.Result
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.True(t, strings.HasPrefix(response.Result, "http://localhost:8080/"))
}

func TestPostHandlerJSON_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, _ := setupTestRouter(ctrl)

	urlService := shortener.NewURLService(nil)
	router.POST("/api/shorten", PostHandlerJSON(urlService, testConfig()))

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("invalid"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === BatchHandler ===

func TestBatchHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)

	repo.EXPECT().Get(gomock.Any()).Return("", errors.New("not found")).Times(2)
	repo.EXPECT().Store(gomock.Any(), "https://one.com", "test-user-123").Return(nil)
	repo.EXPECT().Store(gomock.Any(), "https://two.com", "test-user-123").Return(nil)

	urlService := shortener.NewURLService(repo)
	router.POST("/api/shorten/batch", BatchHandler(urlService, testConfig()))

	batch := []model.URLBatchRequest{
		{CorrelationID: "1", OriginalURL: "https://one.com"},
		{CorrelationID: "2", OriginalURL: "https://two.com"},
	}
	body, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response []model.URLBatchResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Len(t, response, 2)
	assert.Equal(t, "1", response[0].CorrelationID)
	assert.Equal(t, "2", response[1].CorrelationID)
}

func TestBatchHandler_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, _ := setupTestRouter(ctrl)

	urlService := shortener.NewURLService(nil)
	router.POST("/api/shorten/batch", BatchHandler(urlService, testConfig()))

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("invalid"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBatchHandler_EmptyBatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, _ := setupTestRouter(ctrl)

	urlService := shortener.NewURLService(nil)
	router.POST("/api/shorten/batch", BatchHandler(urlService, testConfig()))

	body, _ := json.Marshal([]model.URLBatchRequest{})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response []model.URLBatchResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Empty(t, response)
}

// === GetUserURLsHandler ===

func TestGetUserURLsHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)

	repo.EXPECT().GetUserURLs("test-user-123").Return([]model.URLPair{
		{ShortURL: "abc", OriginalURL: "https://example.com"},
	}, nil)

	urlService := shortener.NewURLService(repo)
	router.GET("/api/user/urls", GetUserURLsHandler(urlService, testConfig()))

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var urls []model.URLPair
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &urls))
	assert.Len(t, urls, 1)
	assert.Equal(t, "http://localhost:8080/abc", urls[0].ShortURL)
}

func TestGetUserURLsHandler_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)

	repo.EXPECT().GetUserURLs("test-user-123").Return([]model.URLPair{}, nil)

	urlService := shortener.NewURLService(repo)
	router.GET("/api/user/urls", GetUserURLsHandler(urlService, testConfig()))

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// === DeleteURLsHandler ===

func TestDeleteURLsHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, repo := setupTestRouter(ctrl)

	repo.EXPECT().DeleteURLs("test-user-123", []string{"abc", "def"})

	urlService := shortener.NewURLService(repo)
	router.DELETE("/api/user/urls", DeleteURLsHandler(urlService))

	body, _ := json.Marshal([]string{"abc", "def"})
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestDeleteURLsHandler_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, _ := setupTestRouter(ctrl)

	urlService := shortener.NewURLService(nil)
	router.DELETE("/api/user/urls", DeleteURLsHandler(urlService))

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader("invalid"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
