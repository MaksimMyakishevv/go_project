package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Глобальная переменная для хранения подключения
var db *gorm.DB

// InitDB инициализирует подключение к базе данных
func InitDB() {
	// Строка подключения: замените на свои данные
	dsn := "host=localhost user=maks password=135798642maks dbname=go port=5432 sslmode=disable"
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	fmt.Println("Подключение к базе данных успешно установлено!")
}

// GetDB возвращает объект подключения к базе данных
func GetDB() *gorm.DB {
	return db
}
