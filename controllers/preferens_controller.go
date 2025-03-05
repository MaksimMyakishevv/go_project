package controllers

import (
	"fmt"
	"net/http"
	"new/dto"
	"new/services"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ErrorResponse — структура для ответа об ошибке
type PreferenceErrorResponse struct {
	Error string `json:"error"`
}

// PreferenceController — контроллер для обработки запросов на предпочтения
type PreferenceController struct {
	Service_prefernse *services.PreferenceService
}

// CreatePreference godoc
// @Summary      Добавить предпочтение
// @Description  Добавляет новое предпочтение для пользователя
// @Tags         preferences
// @Accept       json
// @Produce      json
// @Security BearerAuth
// @Param        input  body      dto.CreatePreferenceDTO  true  "Данные предпочтения"
// @Success      201    {object}  models.Preference
// @Failure      400    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /preferences [post]
func (c *PreferenceController) CreatePreference(ctx *gin.Context) {
	var input dto.CreatePreferenceDTO

	// Проверяем и парсим тело запроса
	if err := ctx.ShouldBindBodyWith(&input, binding.JSON); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Извлекаем userID из контекста
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Преобразуем userID в тип uint
	userIDUint, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse userID"})
		return
	}

	// Вызываем сервис для создания предпочтения
	preference, err := c.Service_prefernse.CreatePreference(userIDUint, input)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем успешный ответ
	ctx.JSON(http.StatusCreated, preference)
}

// GetPreferences godoc
// @Summary      Получить предпочтения
// @Description  Возвращает список предпочтений пользователя
// @Tags         preferences
// @Produce      json
// @Security BearerAuth
// @Success      200  {array}   models.Preference
// @Failure      500  {object}  ErrorResponse
// @Router       /preferences [get]
func (c *PreferenceController) GetPreferences(ctx *gin.Context) {
	userID := ctx.GetUint("userID") // Предполагается, что userID извлекается из middleware

	preferences, err := c.Service_prefernse.GetPreferencesByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, preferences)
}

// DeletePreference godoc
// @Summary      Удалить предпочтение
// @Description  Удаляет предпочтение пользователя по ID
// @Tags         preferences
// @Security BearerAuth
// @Param        id   path      int  true  "ID предпочтения"
// @Success      204  {object}  nil
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /preferences/{id} [delete]
func (c *PreferenceController) DeletePreference(ctx *gin.Context) {
	preferenceID := ctx.Param("id")
	userID := ctx.GetUint("userID") // Предполагается, что userID извлекается из middleware

	if err := c.Service_prefernse.DeletePreference(userID, parseUint(preferenceID)); err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

func parseUint(value string) uint {
	// Преобразование строки в uint с обработкой ошибок
	var parsed uint
	_, err := fmt.Sscanf(value, "%d", &parsed)
	if err != nil {
		return 0
	}
	return parsed
}
