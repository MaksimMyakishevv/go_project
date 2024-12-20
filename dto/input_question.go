package dto


type InputQuestionDTO struct {
	Question    string `json:"question" binding:"required"`
}