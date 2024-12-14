package dto

// RegisterUserDTO — это структура для данных, которые нужно передать при регистрации
type RegisterUserDTO struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}
