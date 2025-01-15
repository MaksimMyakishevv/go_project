package dto


type InputQuestionDTO struct {
	Preferences    string `json:"preferences" binding:"required"`
	Location    string `json:"location" binding:"required"`
}