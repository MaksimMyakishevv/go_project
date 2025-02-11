package controllers

import (
	"net/http"
	services "new/services"
	"new/dto"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type AudioController struct {
	Service *services.AudioService
}

// SaveAudio godoc
// @Summary Сохранить Аудио в БД
// @Description Сохраняет путь до аудиофайла
// @Tags TTS
// @Accept json
// @Produce json
// @Param audio body dto.AudioDTO true "Path data"
// @Success 201 {object} models.Audio "Путь сохранен в БД"
// @Failure 400 {object} ErrorResponse "Invalid input" // Указание структуры ошибки
// @Router /audio [post]
func (c *AudioController) SaveAudio(ctx *gin.Context) {
	var audio dto.AudioDTO

	// Парсим тело запроса
	if err := ctx.ShouldBindBodyWith(&audio, binding.JSON); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	file_path, err := c.Service.SaveAudio(audio)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем успешный ответ
	ctx.JSON(http.StatusCreated, file_path)
}

// GetAllAudio godoc
// @Summary      Получить все аудио
// @Description  Возвращает список путей аудио
// @Tags         TTS
// @Produce      json
// @Success      200  {array}   models.Audio
// @Failure      500  {object}  ErrorResponse
// @Router       /audio [get]
func (c *AudioController) GetAllAudio(ctx *gin.Context) {

	audio, err := c.Service.GetAllAudio()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, audio)
}