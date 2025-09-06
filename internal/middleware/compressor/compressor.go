package compressor

import (
	"compress/gzip"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// Compresser обрабатывает gzip сжатие
func Compresser() gin.HandlerFunc {
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
