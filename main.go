package main

import (
	"net/http"
	"new/controllers"
	"new/database"
	docs "new/docs"
	middleware "new/midellware"
	"new/services"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// Helloworld godoc
// @Summary Returns "helloworld"
// @Description A simple example endpoint that responds with the string "helloworld"
// @Tags Example
// @Accept json
// @Produce json
// @Success 200 {string} string "helloworld"
// @Security BearerAuth
// @Router /helloworld [get]
func Helloworld(c *gin.Context) {
	c.JSON(http.StatusOK, "helloworld")
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

	// Открытые маршруты
	v1 := r.Group("/api")
	{
		v1.POST("/register", RegisController.RegisterUser)
		v1.POST("/login", RegisController.LoginUser)
	}

	// Защищённые маршруты
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/helloworld", Helloworld)
	}

	// Маршрут для Swagger документации
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Запуск сервера
	r.Run(":8080")
}
