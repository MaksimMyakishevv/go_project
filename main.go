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

	// Инициализация сервисов
	RegistService := &services.RegistService{
		DB: database.GetDB(),
	}
	AuthService := &services.AuthService{
		DB: database.GetDB(),
	}

	// Инициализация контроллера
	RegisController := &controllers.RegistController{
		Service_regist: RegistService,
		Service_auth:   AuthService,
	}

	// Настройка маршрутов и Swagger документации
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api"
	v1 := r.Group("/api")
	{
		v1.GET("/helloworld", Helloworld)
		v1.POST("/register", RegisController.RegisterUser)
		v1.POST("/login", RegisController.LoginUser) // добавлен роут для авторизации
	}

	// Маршрут для Swagger документации
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Запуск сервера
	r.Run(":8080")
}
