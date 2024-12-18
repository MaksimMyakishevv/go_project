package services

import (
	"errors"
	"new/dto"
	"new/models"
	"new/utils" // Импортируем новый модуль для работы с JWT

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService — сервис для обработки операций с пользователями
type AuthService struct {
	DB *gorm.DB
}

// AuthenticateUser — проверяет данные пользователя и генерирует JWT токен
func (service *AuthService) AuthenticateUser(loginDTO dto.LoginDTO) (string, error) {
	var user models.User

	// Проверяем, существует ли пользователь с указанным email
	if err := service.DB.Where("email = ?", loginDTO.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("user not found")
		}
		return "", err
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginDTO.Password)); err != nil {
		return "", errors.New("invalid password")
	}

	// Генерация JWT токена с помощью утилиты
	token, err := utils.GenerateJWT(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}
