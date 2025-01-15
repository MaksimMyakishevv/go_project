package controllers

import (
	"net/http"
	services "new/services"

	"new/models"

	"github.com/gin-gonic/gin"
)

type AskLLMController struct {
	Service *services.AskLLMService
}

// AskLLMQuestion godoc
// @Summary Задать вопрос ЛЛМ
// @Description Ввод текста, который будет передан ЛЛМ и возвращение ответа
// @Tags LLM
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body dto.InputQuestionDTO true "Question data"
// @Success 201 {object} models.Question "Вопрос ЛЛМ отправлен"
// @Failure 400 {object} ErrorResponse "Invalid input" // Указание структуры ошибки
// @Router /ask [post]
func (controller *AskLLMController) AskLLMQuestion(c *gin.Context) {
	var question models.Question

	// Bind JSON input to Question model
	if err := c.ShouldBindJSON(&question); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := controller.Service.AskLLMQuestion(question)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to forward item"})
		return
	}

	// Возвращение результата в формате JSON
	c.JSON(http.StatusOK, gin.H{"message": message})
}
