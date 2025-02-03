package dto

// AddPlaceDTO представляет данные для добавления нового места
type AddPlaceDTO struct {
	PlaceName string `json:"place_name" binding:"required"`
}

// ProcessPlacesDTO представляет массив мест для обработки
type ProcessPlacesDTO struct {
	Places []string `json:"places" binding:"required"`
}
