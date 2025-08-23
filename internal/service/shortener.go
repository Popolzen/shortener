package service

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// shortURL создает короткую версию URL
func shortURL(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result strings.Builder
	l := len(charset)

	for range length {
		result.WriteByte(charset[rand.IntN(l)])
	}

	return result.String()
}

// isUniq проверяет что ссылки уже нет
func isUniq(shortURL string, shortURLs map[string]string) bool {
	if _, exsists := shortURLs[shortURL]; !exsists {
		return true
	}

	return false
}

// Функция которая делает ссылку короткой и сохраняет ее в мапу
func Shortener(longURL string, shortURLs map[string]string) (string, error) {
	const length = 6
	const maxAttempts = 1000

	for range maxAttempts {
		su := shortURL(length)
		if isUniq(longURL, shortURLs) {
			shortURLs[su] = longURL
			return su, nil
		}
	}

	return "", fmt.Errorf("не удалось создать уникальную ссылку за %d попыток", maxAttempts)
}
