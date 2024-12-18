package utils

import (
	"errors"
	"fmt"
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
		Subject:   fmt.Sprintf("%d", userID), // Преобразуем userID в строку с использованием fmt.Sprintf
		ExpiresAt: jwt.NewNumericDate(expirationTime),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	// Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен секретным ключом
	return token.SignedString(jwtSecret)
}

// ValidateToken — проверяет JWT токен
func ValidateToken(tokenString string) (jwt.Claims, error) {
	// Парсим и проверяем токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверка типа токена
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Возвращаем claims из токена
	return token.Claims, nil
}
