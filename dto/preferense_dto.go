package dto

// CreatePreferenceDTO используется для передачи данных при создании предпочтения
type CreatePreferenceDTO struct {
	Place    string `json:"place" binding:"required"` // Название места
	Priority int    `json:"priority" binding:"gte=0"` // Приоритет должен быть неотрицательным
}

// UpdatePreferenceDTO используется для передачи данных при обновлении предпочтения
type UpdatePreferenceDTO struct {
	Priority int `json:"priority" binding:"gte=0"` // Новый приоритет предпочтения
}
