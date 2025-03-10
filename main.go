package main

import (
	"net/http"
	backgroundprocesses "new/background_processes"
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
	audioService := &services.AudioService{
		DB: database.GetDB(),
	}
	placeService := &services.PlaceService{ // Добавляем сервис для работы с местами
		DB: database.GetDB(),
	}
	delethistory := &backgroundprocesses.Deletehistory{
		DB: database.GetDB(),
	}
	go delethistory.CleanupOldPlaces()
	askLLMService := &services.AskLLMService{}
	GetAddressesService := &services.GetAddressesService{}

	// Инициализация контроллеров
	regisController := &controllers.RegistController{
		Service_regist: registService,
		Service_auth:   authService,
	}

	askLLMController := &controllers.AskLLMController{
		Service: askLLMService,
	}
	GetAddressesController := &controllers.GetAddressesController{
		Service: GetAddressesService,
	}

	preferenceController := &controllers.PreferenceController{
		Service_prefernse: preferenceService,
	}
	audioController := &controllers.AudioController{
		Service: audioService,
	}
	placeController := &controllers.PlaceController{ // Добавляем контроллер для работы с местами
		Service: placeService,
	}

	// Настройка маршрутов и Swagger документации
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/api"

	// Открытые маршруты
	v1 := r.Group("/api")
	{
		v1.POST("/register", regisController.RegisterUser)
		v1.POST("/login", regisController.LoginUser)
		v1.POST("/ask", askLLMController.AskLLMQuestion) //Эта часть остается в открытом доступе для тестирования
		v1.POST("/addresses", GetAddressesController.GetAddresses)
		v1.POST("/audio", audioController.SaveAudio)  // сохраняет путь в postgres
		v1.POST("/upload", audioController.LoadFile)  //Загрузить аудио в S3
		v1.GET("/audio", audioController.GetAllAudio) //выводит пути аудио из постгреса
		v1.GET("/files", audioController.GetFiles)    //выводит в консоль файлы из бакета
	}

	// Защищённые маршруты
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/preferences", preferenceController.CreatePreference)
		protected.GET("/helloworld", Helloworld)
		protected.GET("/preferences", preferenceController.GetPreferences)
		protected.DELETE("/preferences/:id", preferenceController.DeletePreference)
		protected.GET("/users/history", placeController.GetUserHistory)       // Получение истории запросов пользователя
		protected.POST("/process-json", placeController.ProcessJSON)          // Обработка массива мест
		protected.POST("/cached-response", placeController.GetCachedResponse) // Новый маршрут для получения закешированного ответа
	}

	// Маршрут для Swagger документации
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Запуск сервера
	r.Run(":8080")
}
