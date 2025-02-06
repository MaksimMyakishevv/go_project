package dto

// AddPlaceDTO находит закешированный ответ от ллм модели
type AddPlaceDTO struct {
	PlaceName string `json:"place_name" binding:"required"`
}

// ProcessPlacesDTO представляет массив мест для обработки
type ProcessPlacesDTO struct {
	Places []string `json:"places" binding:"required"`
}
