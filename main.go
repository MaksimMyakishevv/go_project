package main

import (
	"net/http"
	"new/controllers"
	"new/database"
	docs "new/docs"
	"new/services"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @BasePath /api

// Helloworld godoc
// @Summary Returns "helloworld"
// @Description A simple example endpoint that responds with the string "helloworld"
// @Tags Example
// @Accept json
// @Produce json
// @Success 200 {string} string "helloworld"
// @Router /helloworld [get]
func Helloworld(g *gin.Context) {
	g.JSON(http.StatusOK, "helloworld")
}

func main() {
	// Инициализация подключения к базе данных

	database.InitDB()
	// Инициализация сервиса
	authService := &services.AuthService{
		DB: database.GetDB(),
	}

	// Инициализация контроллера
	authController := &controllers.AuthController{
		Service: authService,
	}

	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api"
	v1 := r.Group("/api")
	{
		v1.GET("/helloworld", Helloworld)
		v1.POST("/register", authController.RegisterUser)
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	r.Run(":8080")

}
