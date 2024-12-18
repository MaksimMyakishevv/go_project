package services

import (
	"errors"
	"new/dto"
	"new/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService — сервис для обработки операций с пользователями
type RegistService struct {
	DB *gorm.DB
}

// RegisterUser регистрирует нового пользователя
func (service *RegistService) RegisterUser(userDTO dto.RegisterUserDTO) (*models.User, error) {
	// Проверяем, существует ли пользователь с таким же username или email
	var user models.User
	if err := service.DB.Where("username = ?", userDTO.Username).First(&user).Error; err == nil {
		return nil, errors.New("username already taken")
	}
	if err := service.DB.Where("email = ?", userDTO.Email).First(&user).Error; err == nil {
		return nil, errors.New("email already taken")
	}

	// Хэшируем пароль перед сохранением
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userDTO.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Создаем нового пользователя с хэшированным паролем
	newUser := models.User{
		Username: userDTO.Username,
		Password: string(hashedPassword), // Храним хэш пароля
		Email:    userDTO.Email,
	}

	// Сохраняем нового пользователя в базу данных
	if err := service.DB.Create(&newUser).Error; err != nil {
		return nil, err
	}
	return &newUser, nil
}
