package subnet

import (
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TrustedSubnetMiddleware проверяет, что IP-адрес клиента входит в доверенную подсеть.
//
// Параметры:
//   - trustedSubnet: строковое представление CIDR (например, "192.168.1.0/24")
//
// Логика:
//   - Если trustedSubnet пустая строка → всегда возвращает 403 Forbidden
//   - Извлекает IP из заголовка X-Real-IP
//   - Проверяет вхождение IP в указанную подсеть
//   - Если IP не входит в подсеть → возвращает 403 Forbidden
//
// Пример использования:
//
//	internal := r.Group("/api/internal")
//	internal.Use(subnet.TrustedSubnetMiddleware("192.168.1.0/24"))
//	{
//	    internal.GET("/stats", handler.StatsHandler(service))
//	}
func TrustedSubnetMiddleware(trustedSubnet string) gin.HandlerFunc {
	// Если подсеть не указана, запрещаем доступ всем
	if trustedSubnet == "" {
		return func(c *gin.Context) {
			log.Println("Доступ запрещен: доверенная подсеть не настроена")
			c.AbortWithStatus(http.StatusForbidden)
		}
	}

	// Парсим CIDR один раз при создании middleware
	_, ipNet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		log.Printf("Ошибка парсинга CIDR '%s': %v", trustedSubnet, err)
		// Если CIDR невалиден, запрещаем доступ всем
		return func(c *gin.Context) {
			log.Printf("Доступ запрещен: невалидный CIDR '%s'", trustedSubnet)
			c.AbortWithStatus(http.StatusForbidden)
		}
	}

	return func(c *gin.Context) {
		// Извлекаем IP из заголовка X-Real-IP
		realIP := c.GetHeader("X-Real-IP")
		if realIP == "" {
			log.Println("Доступ запрещен: заголовок X-Real-IP отсутствует")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Парсим IP-адрес
		ip := net.ParseIP(realIP)
		if ip == nil {
			log.Printf("Доступ запрещен: невалидный IP '%s'", realIP)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Проверяем вхождение IP в доверенную подсеть
		if !ipNet.Contains(ip) {
			log.Printf("Доступ запрещен: IP %s не входит в доверенную подсеть %s", realIP, trustedSubnet)
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		log.Printf("Доступ разрешен: IP %s входит в доверенную подсеть %s", realIP, trustedSubnet)
		c.Next()
	}
}
