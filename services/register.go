package services

import (
	"errors"
	"new/dto"
	"new/models"

	"gorm.io/gorm"
)

// AuthService — сервис для обработки операций с пользователями
type AuthService struct {
	DB *gorm.DB
}

// RegisterUser регистрирует нового пользователя
func (service *AuthService) RegisterUser(userDTO dto.RegisterUserDTO) (*models.User, error) {
	// Проверяем, существует ли пользователь с таким же username или email
	var user models.User
	if err := service.DB.Where("username = ?", userDTO.Username).First(&user).Error; err == nil {
		return nil, errors.New("username already taken")
	}
	if err := service.DB.Where("email = ?", userDTO.Email).First(&user).Error; err == nil {
		return nil, errors.New("email already taken")
	}

	// Создаем нового пользователя
	newUser := models.User{
		Username: userDTO.Username,
		Password: userDTO.Password, // На практике пароли должны хешироваться
		Email:    userDTO.Email,
	}

	if err := service.DB.Create(&newUser).Error; err != nil {
		return nil, err
	}
	return &newUser, nil
}
