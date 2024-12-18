package middleware

import (
	"net/http"
	"new/utils"

	//"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware — middleware для проверки JWT токена
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из заголовков
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}

		// Токен в заголовке должен быть в формате: "Bearer <token>"
		// parts := strings.Split(authHeader, " ")
		// if len(parts) != 2 || parts[0] != "Bearer" {
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
		// 	c.Abort()
		// 	return
		// }

		// Проверяем токен с помощью утилиты
		_, err := utils.ValidateToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Если токен валиден, продолжаем выполнение запроса
		c.Next()
	}
}
