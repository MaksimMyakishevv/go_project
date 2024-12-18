package controllers

import (
	"net/http"
	"new/dto"
	"new/services"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// AuthController — контроллер для обработки запросов авторизации
type AuthController struct {
	Service *services.AuthService
}

// LoginUser — метод контроллера для авторизации пользователя
// @Summary Login user
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param login body dto.LoginDTO true "User credentials"
// @Success 200 {object} map[string]string "JWT token"
// @Failure 400 {object} ErrorResponse "Invalid input"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Router /login [post]
func (controller *AuthController) LoginUser(c *gin.Context) {
	// Валидация входных данных
	var loginDTO dto.LoginDTO
	if err := c.ShouldBindBodyWith(&loginDTO, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Аутентификация пользователя
	token, err := controller.Service.AuthenticateUser(loginDTO)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	// Возврат JWT токена
	c.JSON(http.StatusOK, gin.H{"token": token})
}
