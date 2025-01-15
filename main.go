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
	registService := &services.RegistService{
		DB: database.GetDB(),
	}
	authService := &services.AuthService{
		DB: database.GetDB(),
	}
	preferenceService := &services.PreferenceService{
		DB: database.GetDB(),
	}
	askLLMService := &services.AskLLMService{}

	// Инициализация контроллеров
	regisController := &controllers.RegistController{
		Service_regist: registService,
		Service_auth:   authService,
	}

	askLLMController := &controllers.AskLLMController{
		Service: askLLMService,
	}

	preferenceController := &controllers.PreferenceController{
		Service_prefernse: preferenceService,
	}

	// Настройка маршрутов и Swagger документации
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api"

	// Открытые маршруты
	v1 := r.Group("/api")
	{
		v1.POST("/register", regisController.RegisterUser)
		v1.POST("/login", regisController.LoginUser) // добавлен роут для авторизации

	}

	// Защищённые маршруты
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/preferences", preferenceController.CreatePreference)
		protected.GET("/helloworld", Helloworld)
		protected.POST("/ask", askLLMController.AskLLMQuestion)
		protected.GET("/preferences", preferenceController.GetPreferences)
		protected.PUT("/preferences/:id", preferenceController.UpdatePreference)
		protected.DELETE("/preferences/:id", preferenceController.DeletePreference)
	}

	// Маршрут для Swagger документации
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Запуск сервера
	r.Run(":8080")
}
