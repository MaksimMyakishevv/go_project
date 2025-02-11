package dto

type AudioDTO struct {
	Path    string `json:"path" binding:"required"`
}