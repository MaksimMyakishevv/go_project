package dto


type InputQuestionDTO struct {
	Message    string `json:"message" binding:"required"`
}