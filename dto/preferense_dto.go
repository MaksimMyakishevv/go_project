package dto

// CreatePreferenceDTO используется для передачи данных при создании предпочтения
type CreatePreferenceDTO struct {
	ListPreferenceID uint `json:"list_preference_id" binding:"required"`
}
