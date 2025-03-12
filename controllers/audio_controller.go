package controllers

import (
	"context"
	"fmt"
	"net/http"
	"new/models"
	services "new/services"
	"os"

	"github.com/gin-gonic/gin"
)

type AudioController struct {
	Service *services.AudioService
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
	audio := "Проверь консоль сервера, туда вывело"

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
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Ошибка при получении файла: %s", err.Error()),
		})
		return
	}
	defer file.Close()

	// 2. Создаем модель UploadedFile
	uploadedFile := models.UploadedFile{
		File:     file,
		Filename: header.Filename,
		Size:     header.Size,
	}

	// 3. Вызываем сервис для загрузки файла
	ctx := context.Background()
	fileURL, err := ac.Service.LoadFile(uploadedFile, ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Ошибка при загрузке файла: %s", err.Error()),
		})
		return
	}

	// 4. Возвращаем успешный ответ с ссылкой на файл
	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("Файл '%s' успешно загружен в bucket '%s'", uploadedFile.Filename, bucketName),
		"file_url": fileURL,
	})
}
