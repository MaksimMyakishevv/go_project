package controllers

import (
	"net/http"
	"new/services"
	"new/models"

	"github.com/gin-gonic/gin"
)

type GetAddressesController struct {
	Service *services.GetAddressesService
}

// GetAddressesList godoc
// @Summary Получить список адресов
// @Description Заглушка, возвращающая и списка адресов - строку
// @Tags LLM
// @Accept json
// @Produce json

// @Success 201 {object} models.Question "Массис отправлен"
// @Failure 400 {object} ErrorResponse "Invalid input" // Указание структуры ошибки
// @Router /addresses [post]
func (controller *GetAddressesController) GetAddresses(c *gin.Context) {

	var addresses models.Addresses

    // Bind the JSON input to the variable
    if err := c.ShouldBindJSON(&addresses); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Use the service to process the input
    response, err := controller.Service.GetAddresses(addresses)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // Return the response string
    c.JSON(http.StatusOK, gin.H{"answer": response})

}
