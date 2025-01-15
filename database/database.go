package database

import (
	"fmt"
	"log"
	"new/models"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Глобальная переменная для хранения подключения
var db *gorm.DB

// InitDB инициализирует подключение к базе данных PostgreSQL

func InitDB() {
	time.Sleep(5 * time.Second) // Задержка на 5 секунд перед подключением для докеров нужна
	// Строка подключения: замени на свои данные (например, параметры подключения к PostgreSQL)
	dsn := "host=localhost user=root password=password dbname=demodb port=5433 sslmode=disable"

	// Открытие подключения к базе данных PostgreSQL
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	fmt.Println("Подключение к базе данных успешно установлено!")
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
