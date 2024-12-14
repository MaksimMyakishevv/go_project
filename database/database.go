package database

import (
	"fmt"
	"log"
	"new/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Глобальная переменная для хранения подключения
var db *gorm.DB

// InitDB инициализирует подключение к базе данных PostgreSQL
// Используйте свою строку подключения к базе данных
func InitDB() {
	// Строка подключения: замените на свои данные (например, параметры подключения к PostgreSQL)
	dsn := "host=localhost user=postgres password=postgres dbname=go port=5432 sslmode=disable"

	// Открытие подключения к базе данных PostgreSQL
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err) // Логируем ошибку подключения
	}

	fmt.Println("Подключение к базе данных успешно установлено!") // Выводим сообщение об успешном подключении
	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}
}

// GetDB возвращает объект подключения к базе данных
// Используется для получения доступа к базе данных в других частях приложения
func GetDB() *gorm.DB {
	return db
}
