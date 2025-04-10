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

// GenerateAudio godoc
// @Summary      Сгенерировать аудио
// @Description  Генерирует аудиофайл в формате MP3
// @Tags         audio
// @Accept       json
// @Produce      octet-stream
// @Param        input body dto.AudioDTO true "Текст для генерации аудио"
// @Success      200  {string}  binary  "Бинарные данные аудиофайла"
// @Failure      400  {object}  PlaceErrorResponse
// @Failure      500  {object}  PlaceErrorResponse
// @Router       /audio/generate [post]
func (c *PlaceController) GenerateAudioFromText(ctx *gin.Context) {
	var request dto.AudioDTO

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, PlaceErrorResponse{Error: "Некорректный запрос: " + err.Error()})
		return
	}

	if request.Message == "" {
		ctx.JSON(http.StatusBadRequest, PlaceErrorResponse{Error: "Поле 'message' обязательно"})
		return
	}

	audioData, err := c.Service.AudioGenerate(request.Message)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PlaceErrorResponse{Error: "Ошибка генерации: " + err.Error()})
		return
	}

	// Отправляем аудио как бинарный поток
	ctx.JSON(http.StatusOK, audioData)
}

// ProcessJSON godoc
// @Summary      Обработать JSON-файл с местами
// @Description  Обрабатывает JSON-файл с объектами мест и отправляет их на нейросеть
// @Tags         places
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input  body      dto.ProcessPlacesDTO  true  "JSON-файл с местами"
// @Success      200    {array}   map[string]interface{}
// @Failure      400    {object}  PlaceErrorResponse
// @Failure      500    {object}  PlaceErrorResponse
// @Router       /process-json [post]
func (c *PlaceController) ProcessJSON(ctx *gin.Context) {
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
	// Вызываем сервис для обработки JSON-файла
	results, err := c.Service.ProcessJSON(userIDUint, input.JSONData)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PlaceErrorResponse{Error: err.Error()})
		return
	}
	// Возвращаем результаты
	ctx.JSON(http.StatusOK, results)
}

// GetCachedResponse godoc
// @Summary      Получить закешированный ответ
// @Description  Возвращает закешированный ответ из Redis для конкретного пользователя и места
// @Tags         places
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        input  body      dto.AddPlaceDTO  true  "Запрос с названием места"
// @Success      200    {object}  PlaceErrorResponse "Верный формат"
// @Failure      400    {object}  PlaceErrorResponse "Неверный формат запроса"
// @Failure      401    {object}  PlaceErrorResponse "Пользователь не авторизован"
// @Failure      404    {object}  PlaceErrorResponse "Ответ не найден в кеше"
// @Failure      500    {object}  PlaceErrorResponse "Ошибка сервера"
// @Router       /cached-response [post]
func (c *PlaceController) GetCachedResponse(ctx *gin.Context) {
	var input dto.AddPlaceDTO
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	// Проверяем, что название места не пустое
	if input.PlaceName == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Поле 'place_name' обязательно"})
		return
	}

	// Извлекаем userID из JWT-токена
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

	// Логируем запрос
	println("Получен запрос на поиск закешированного ответа для пользователя:", userIDUint, "и места:", input.PlaceName)

	// Получаем закешированный ответ
	response, err := c.Service.GetCachedResponse(userIDUint, input.PlaceName)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Возвращаем успешный ответ
	ctx.JSON(http.StatusOK, gin.H{"response": response})
}

// ProcessJSONNoAuth godoc
// @Summary      Обработать JSON-файл с местами
// @Description  Обрабатывает JSON-файл с объектами мест и отправляет их на нейросеть
// @Tags         places
// @Accept       json
// @Produce      json
// @Param        input  body      dto.ProcessPlacesDTO  true  "JSON-файл с местами"
// @Success      200    {array}   map[string]interface{}
// @Failure      400    {object}  PlaceErrorResponse
// @Failure      500    {object}  PlaceErrorResponse
// @Router       /process-json-noauth [post]
func (c *PlaceController) ProcessJSONNoAuth(ctx *gin.Context) {
	var input dto.ProcessPlacesDTO
	// Проверяем и парсим тело запроса
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, PlaceErrorResponse{Error: err.Error()})
		return
	}

	// Вызываем сервис для обработки JSON-файла
	results, err := c.Service.ProcessJSONNoAuth(input.JSONData)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, PlaceErrorResponse{Error: err.Error()})
		return
	}
	// Возвращаем результаты
	ctx.JSON(http.StatusOK, results)
}
