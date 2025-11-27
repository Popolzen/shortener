package compressor

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Compresser())
	return r
}

func gzipCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func gzipDecompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func TestCompresser_DecompressRequest(t *testing.T) {
	router := setupRouter()

	var receivedBody string
	router.POST("/test", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		receivedBody = string(body)
		c.String(http.StatusOK, "ok")
	})

	originalBody := "Hello, compressed world!"
	compressedBody := gzipCompress([]byte(originalBody))

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(compressedBody))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, originalBody, receivedBody)
}

func TestCompresser_CompressJSONResponse(t *testing.T) {
	router := setupRouter()

	router.GET("/json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, `{"message":"hello"}`)
	})

	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

	decompressed, err := gzipDecompress(w.Body.Bytes())
	require.NoError(t, err)
	assert.Equal(t, `{"message":"hello"}`, string(decompressed))
}

func TestCompresser_CompressHTMLResponse(t *testing.T) {
	router := setupRouter()

	router.GET("/html", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, "<html><body>Hello</body></html>")
	})

	req := httptest.NewRequest(http.MethodGet, "/html", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

	decompressed, err := gzipDecompress(w.Body.Bytes())
	require.NoError(t, err)
	assert.Contains(t, string(decompressed), "<html>")
}

func TestCompresser_NoCompressWithoutAcceptEncoding(t *testing.T) {
	router := setupRouter()

	router.GET("/plain", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, `{"data":"test"}`)
	})

	req := httptest.NewRequest(http.MethodGet, "/plain", nil)
	// Не устанавливаем Accept-Encoding
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Equal(t, `{"data":"test"}`, w.Body.String())
}

func TestCompresser_NoCompressTextPlain(t *testing.T) {
	router := setupRouter()

	router.GET("/text", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, "plain text")
	})

	req := httptest.NewRequest(http.MethodGet, "/text", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// text/plain не сжимается по твоей логике
	assert.Empty(t, w.Header().Get("Content-Encoding"))
}

func TestCompresser_InvalidGzipRequest(t *testing.T) {
	router := setupRouter()

	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("not gzip data")))
	req.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCompresser_RoundTrip(t *testing.T) {
	router := setupRouter()

	router.POST("/echo", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Header("Content-Type", "application/json")
		c.String(http.StatusOK, string(body))
	})

	originalData := `{"test":"roundtrip data"}`
	compressedReq := gzipCompress([]byte(originalData))

	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(compressedReq))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	decompressed, err := gzipDecompress(w.Body.Bytes())
	require.NoError(t, err)
	assert.Equal(t, originalData, string(decompressed))
}
