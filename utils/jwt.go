package utils

import (
	"errors"
	"fmt"
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

// ExtractUserIDFromToken — извлекает userID из JWT токена
func ExtractUserIDFromToken(tokenString string) (uint, error) {
	// Парсим токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверка метода подписи токена
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method") // Ошибка при некорректном методе подписи
		}
		return jwtSecret, nil // Возвращаем секретный ключ для проверки подписи
	})
	if err != nil || !token.Valid {
		return 0, errors.New("invalid token") // Ошибка, если токен невалидный
	}

	// Извлекаем claims из токена
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token claims") // Ошибка при некорректных claims
	}

	// Извлекаем userID из claims
	userIDStr, ok := claims["sub"].(string) // Достаем поле "sub" (userID)
	if !ok {
		return 0, errors.New("invalid userID in token claims") // Ошибка, если поле "sub" отсутствует или неверного типа
	}

	// Преобразуем userID из строки в uint
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return 0, errors.New("failed to parse userID") // Ошибка при преобразовании userID
	}

	return uint(userID), nil // Возвращаем userID как uint
}
