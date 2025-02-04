package database

import (
	"fmt"
	"log"
	"new/models"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Глобальная переменная для хранения подключения
var db *gorm.DB

// InitDB инициализирует подключение к базе данных PostgreSQL

func InitDB() {

	errr := godotenv.Load()
	if errr != nil {
		log.Fatalf("Ошибка загрузки .env файла: %v", errr)
	}
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	time.Sleep(5 * time.Second) // Задержка на 5 секунд перед подключением для докеров нужна
	// Строка подключения: замени на свои данные (например, параметры подключения к PostgreSQL)
	dsn := "host=" + dbHost +
		" user=" + dbUser +
		" password=" + dbPassword +
		" dbname=" + dbName +
		" port=" + dbPort +
		" sslmode=disable"

	// Открытие подключения к базе данных PostgreSQL
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	fmt.Println("Подключение к базе данных успешно установлено!") // Выводим сообщение об успешном подключении
	err = db.AutoMigrate(&models.User{}, &models.Preference{}, &models.Place{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}
}

// GetDB возвращает объект подключения к базе данных
// Используется для получения доступа к базе данных в других частях приложения
func GetDB() *gorm.DB {
	return db
}
