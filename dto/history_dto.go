package dto

// AddPlaceDTO находит закешированный ответ от ллм модели
type AddPlaceDTO struct {
	PlaceName string `json:"place_name" binding:"required"`
}
type OSMObject struct {
	ID      int64
	Type    string
	Tags    map[string]string
	Lat     float64    // Только для node
	Lon     float64    // Только для node
	Nodes   []int64    // Только для way
	Members []struct { // Только для relation
		Type string
		Ref  int64
		Role string
	} `json:"members,omitempty"`
}

// ProcessPlacesDTO представляет массив мест для обработки
type ProcessPlacesDTO struct {
	JSONData []OSMObject `json:"json_data" binding:"required"`
}
