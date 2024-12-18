package utils

import (
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWT секретный ключ
var jwtSecret = []byte("your_secret_key")

// GenerateJWT — генерирует JWT токен для пользователя
func GenerateJWT(userID uint) (string, error) {
	// Определяем срок действия токена (например, 24 часа)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Создаем claims токена
	claims := &jwt.RegisteredClaims{
		Subject:   strconv.Itoa(int(userID)), // Преобразуем userID в строку
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	// Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен секретным ключом
	return token.SignedString(jwtSecret)
}
