package controllers

import (
	"fmt"
	"net/http"
	"new/dto"
	"new/models"
	services "new/services"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type AudioController struct {
	Service *services.AudioService
}

// SaveAudio godoc
// @Summary Сохранить Аудио в БД postgres
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
// @Summary      Получить все пути аудио
// @Description  Возвращает список путей аудио из БД
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

// GetAllAudio godoc
// @Summary      Получить все файлы в бакете
// @Description  Возвращает список информации о файлах в бакете в ТЕРМИНАЛ
// @Tags         TTS
// @Produce      json
// @Failure      500  {object}  ErrorResponse
// @Router       /files [get]
func (c *AudioController) GetFiles(ctx *gin.Context) {

	err := c.Service.GetFiles()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	audio:="Проверь консоль сервера, туда вывело"

	ctx.JSON(http.StatusOK, audio)
}


// UploadFile godoc
// @Summary Загрузить файл в Object Storage
// @Description Загружает файл в Object Storage Яндекса
// @Tags TTS
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Success 200 {array}   map[string]interface{}
// @Failure 400 {object} ErrorResponse "Invalid input or upload failed"
// @Router /upload [post]
func (ac *AudioController) LoadFile(c *gin.Context) {

	bucketName := os.Getenv("BUCKET_NAME")
	
	// 1. Получение файла из запроса
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Ошибка при получении файла: %s", err.Error()))
		return
	}
	defer file.Close()

	// 2. Создаем модель UploadedFile
	uploadedFile := models.UploadedFile{
		File: file,
		Filename: header.Filename,
		Size: header.Size,
	}


	// 3. Вызываем сервис для загрузки файла
	err = ac.Service.LoadFile(uploadedFile, c)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Ошибка при загрузке файла: %s", err.Error()))
		return
	}

	c.String(http.StatusOK, fmt.Sprintf("Файл '%s' успешно загружен в bucket '%s'", uploadedFile.Filename, bucketName))
}