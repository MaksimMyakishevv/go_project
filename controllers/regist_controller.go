package controllers

import (
	"net/http"
	"new/dto"
	services "new/services"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ErrorResponse — структура для ответа об ошибке
type ErrorResponse struct {
	Error string `json:"error"` // Поле с описанием ошибки
}

// AuthController — контроллер для обработки запросов на регистрацию
type RegistController struct {
	Service *services.RegistService
}

// RegisterUser godoc
// @Summary Register new user
// @Description Register a new user by providing username, password, and email
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dto.RegisterUserDTO true "User data"
// @Success 201 {object} models.User "Successfully created user"
// @Failure 400 {object} ErrorResponse "Invalid input" // Указание структуры ошибки
// @Router /register [post]
func (controller *RegistController) RegisterUser(c *gin.Context) {
	var userDTO dto.RegisterUserDTO
	if err := c.ShouldBindBodyWith(&userDTO, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()}) // Использование структуры ошибки
		return
	}

	user, err := controller.Service.RegisterUser(userDTO)
	if err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()}) // Использование структуры ошибки
		return
	}

	c.JSON(http.StatusCreated, user)
}
