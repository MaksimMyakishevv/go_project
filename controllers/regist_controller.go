package controllers

import (
	"net/http"
	"new/dto"
	"new/services"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ErrorResponse — структура для ответа об ошибке
type ErrorResponse struct {
	Error string `json:"error"`
}

// TokenResponse — структура для ответа с токеном
type TokenResponse struct {
	Token string `json:"token"`
}

// RegistController — контроллер для обработки запросов на регистрацию и вход
type RegistController struct {
	Service_regist *services.RegistService
	Service_auth   *services.AuthService
}

// RegisterUser godoc
// @Summary Register new user
// @Description Register a new user by providing username, password, and email
// @Tags auth
// @Accept json
// @Produce json
// @Param user body dto.RegisterUserDTO true "User data"
// @Success 201 {object} models.User "Successfully created user"
// @Failure 400 {object} ErrorResponse "Invalid input"
// @Failure 409 {object} ErrorResponse "Conflict - user already exists"
// @Router /register [post]
func (controller *RegistController) RegisterUser(c *gin.Context) {
	var userDTO dto.RegisterUserDTO
	if err := c.ShouldBindBodyWith(&userDTO, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	user, err := controller.Service_regist.RegisterUser(userDTO)
	if err != nil {
		c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// LoginUser godoc
// @Summary Login user and return JWT token
// @Description Login a user by providing email and password, and return a JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param login body dto.LoginDTO true "User login data"
// @Success 200 {object} TokenResponse "JWT token"
// @Failure 400 {object} ErrorResponse "Invalid input"
// @Failure 401 {object} ErrorResponse "Unauthorized - invalid credentials"
// @Router /login [post]
func (controller *RegistController) LoginUser(c *gin.Context) {
	var loginDTO dto.LoginDTO
	if err := c.ShouldBindBodyWith(&loginDTO, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	token, err := controller.Service_auth.AuthenticateUser(loginDTO)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	// Теперь возвращаем TokenResponse вместо gin.H
	c.JSON(http.StatusOK, TokenResponse{Token: token})
}
