package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const secretKey = "guess_me"

func validateCookie(cookieValue string) (string, bool) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return "", false
	}
	userID, signature := parts[0], parts[1]

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(userID))
	expectedSignature := mac.Sum(nil)

	// Декодируем полученную подпись из base64
	receivedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return "", false
	}

	// Сравниваем байты HMAC
	return userID, hmac.Equal(receivedSignature, expectedSignature)
}

// signUserID подписывает UserID с использованием HMAC-SHA256
func signUserID(userID string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(userID))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return userID + "." + signature
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var userID string
		var isValid bool
		var hadCookie bool

		cookie, err := c.Cookie("user_id")
		hadCookie = (err == nil && cookie != "")

		if !hadCookie {
			userID = uuid.New().String()
			isValid = false
		} else {
			userID, isValid = validateCookie(cookie)
			if !isValid {
				userID = uuid.New().String()
			}
		}

		signedValue := signUserID(userID)
		c.SetCookie("user_id", signedValue, 3600*24*30, "/", "", true, true)

		// Передаем всю нужную информацию в контекст
		c.Set("user_id", userID)
		c.Set("cookie_was_valid", isValid)
		c.Set("had_cookie", hadCookie)

		c.Next()
	}
}
