package interceptors

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/Popolzen/shortener/internal/config"
	"github.com/Popolzen/shortener/internal/model"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const userIDKey model.ContextKey = "user_id"

// UnaryInterceptor создает interceptor для аутентификации
func UnaryInterceptor(cfg *config.Config) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Получаем metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Извлекаем authorization header
		authHeaders := md.Get("authorization")
		var userID string

		if len(authHeaders) == 0 || authHeaders[0] == "" {
			// Нет токена - создаем нового пользователя
			userID = uuid.New().String()
		} else {
			// Валидируем токен
			token := authHeaders[0]
			validatedUserID, valid := validateToken(token, cfg.SecretKey)
			if !valid {
				// Невалидный токен - создаем нового пользователя
				userID = uuid.New().String()
			} else {
				userID = validatedUserID
			}
		}

		// Добавляем userID в контекст
		ctx = context.WithValue(ctx, userIDKey, userID)

		// Добавляем новый токен в response metadata
		newToken := signUserID(userID, cfg.SecretKey)
		header := metadata.Pairs("authorization", newToken)
		grpc.SetHeader(ctx, header)

		// Вызываем handler
		return handler(ctx, req)
	}
}

// validateToken проверяет HMAC токен и возвращает userID
func validateToken(token, secretKey string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", false
	}

	userID, signature := parts[0], parts[1]

	// Вычисляем ожидаемую подпись
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(userID))
	expectedSignature := mac.Sum(nil)

	// Декодируем полученную подпись
	receivedSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return "", false
	}

	// Сравниваем
	if !hmac.Equal(receivedSignature, expectedSignature) {
		return "", false
	}

	return userID, true
}

// signUserID создает HMAC токен для userID
func signUserID(userID, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(userID))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return userID + "." + signature
}
