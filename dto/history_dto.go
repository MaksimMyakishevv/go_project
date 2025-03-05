package dto

// AddPlaceDTO находит закешированный ответ от ллм модели
type AddPlaceDTO struct {
	PlaceName string `json:"place_name" binding:"required"`
}
type OSMObject struct {
	Type  string            `json:"type"`
	ID    int               `json:"id"`
	Nodes []int             `json:"nodes"`
	Tags  map[string]string `json:"tags"`
}

// ProcessPlacesDTO представляет массив мест для обработки
type ProcessPlacesDTO struct {
	JSONData []OSMObject `json:"json_data" binding:"required"`
}
