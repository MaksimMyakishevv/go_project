package controllers

import (
	"net/http"
	"new/dto"
	"new/services"

	"github.com/gin-gonic/gin"
)

// ErrorResponse — структура для ответа об ошибке
type PlaceErrorResponse struct {
	Error string `json:"error"`
}

// PlaceController — контроллер для обработки запросов на места
type PlaceController struct {
	Service *services.PlaceService
}

// GetUserHistory godoc
// @Summary      Получить историю запросов
// @Description  Возвращает список мест, связанных с пользователем
// @Tags         places
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}   models.Place
// @Failure      500  {object}  PlaceErrorResponse
// @Router       /users/history [get]
func (c *PlaceController) GetUserHistory(ctx *gin.Context) {
	userID := ctx.GetUint("userID") // Предполагается, что userID извлекается из middleware
	history, err := c.Service.GetUserHistory(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PlaceErrorResponse{Error: err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, history)
}

// ProcessPlaces godoc
// @Summary      Обработать массив мест
// @Description  Последовательно обрабатывает массив мест и отправляет их на нейросеть
// @Tags         places
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input  body      dto.ProcessPlacesDTO  true  "Массив мест"
// @Success      200    {array}   map[string]interface{}
// @Failure      400    {object}  PlaceErrorResponse
// @Failure      500    {object}  PlaceErrorResponse
// @Router       /process-places [post]
func (c *PlaceController) ProcessPlaces(ctx *gin.Context) {
	var input dto.ProcessPlacesDTO
	// Проверяем и парсим тело запроса
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, PlaceErrorResponse{Error: err.Error()})
		return
	}
	// Извлекаем userID из контекста
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, PlaceErrorResponse{Error: "User not authenticated"})
		return
	}
	// Преобразуем userID в тип uint
	userIDUint, ok := userID.(uint)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, PlaceErrorResponse{Error: "Failed to parse userID"})
		return
	}
	// Вызываем сервис для обработки массива мест
	results, err := c.Service.ProcessPlaces(userIDUint, input.Places)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PlaceErrorResponse{Error: err.Error()})
		return
	}
	// Возвращаем результаты
	ctx.JSON(http.StatusOK, results)
}
