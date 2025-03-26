package dto

type AudioDTO struct {
	Message    string `json:"message" binding:"required"`
}