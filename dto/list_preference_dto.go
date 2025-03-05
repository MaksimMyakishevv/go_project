package dto

type CreateListPreferenceDTO struct {
	Place string `json:"place" binding:"required,"`
}
