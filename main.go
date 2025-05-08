package main

import (
	"log"

	backgroundprocesses "new/background_processes"
	"new/controllers"
	"new/database"
	docs "new/docs"
	middleware "new/middleware"
	"new/services"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	// Инициализация подключения к базе данных
	database.InitDB()
	database.InitRedis()

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
	placeService := &services.PlaceService{
		DB: database.GetDB(),
	}
	delethistory := &backgroundprocesses.Deletehistory{
		DB: database.GetDB(),
	}
	go delethistory.CleanupOldPlaces()
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
	placeController := &controllers.PlaceController{
		Service: placeService,
	}

	// Создаём WebSocket-обработчик
	wsHandler := services.NewWebSocketHandler(placeService)

	// Настройка маршрутов и Swagger документации
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api"

	// Открытые маршруты
	v1 := r.Group("/api")
	{
		v1.POST("/register", regisController.RegisterUser)
		v1.POST("/login", regisController.LoginUser)
		v1.POST("/ask", askLLMController.AskLLMQuestion)                     //Эта часть остается в открытом доступе для тестирования
		v1.POST("/audio/generate", placeController.GenerateAudioFromText)    //Генерация аудио из текста
		v1.POST("/process-json-noauth", placeController.ProcessJSONNoAuth)   //Обработка массива данных без необходимости регистрироваться
		v1.POST("/process-json-mistral", placeController.ProcessJSONMistral) //Реальная ЛЛМ Mistral
	}

	// Защищённые маршруты
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/preferences", preferenceController.CreatePreference)
		protected.GET("/preferences", preferenceController.GetPreferences)
		protected.DELETE("/preferences/:id", preferenceController.DeletePreference)
		protected.GET("/users/history", placeController.GetUserHistory)
		protected.POST("/process-json", placeController.ProcessJSON)
		protected.POST("/cached-response", placeController.GetCachedResponse)
	}

	// Маршрут для Swagger документации
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Добавляем WebSocket маршрут
	r.GET("/ws", func(c *gin.Context) {
		wsHandler.HandleWebSocket(c.Writer, c.Request)
	})

	// Запуск сервера
	log.Println("Сервер запущен на :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
