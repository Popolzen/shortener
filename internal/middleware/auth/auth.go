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

// validateCookie валидирует подписанную куки и возвращает userID, если валидна.
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

// getOrCreateUserID извлекает userID из куки, если валидна, или генерирует новый.
func getOrCreateUserID(c *gin.Context) (string, bool, bool) {
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

	return userID, isValid, hadCookie
}

// setSignedCookie подписывает userID и устанавливает куки в ответе.
func setSignedCookie(c *gin.Context, userID string) {
	signedValue := signUserID(userID)
	c.SetCookie("user_id", signedValue, 3600*24*30, "/", "", false, true)
}

// AuthMiddleware - middleware для обработки аутентификации пользователя через куки.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, isValid, hadCookie := getOrCreateUserID(c)
		setSignedCookie(c, userID)

		c.Set("user_id", userID)
		c.Set("cookie_was_valid", isValid)
		c.Set("had_cookie", hadCookie)

		c.Next()
	}
}
