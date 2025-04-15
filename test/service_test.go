package test

import (
	"context"
	"fmt"
	"new/database"
	"new/services"
	"os"
	"testing"
)

func generatePlaces(count int, offset int) []map[string]string {
	places := make([]map[string]string, count)
	for i := 0; i < count; i++ {
		placeName := fmt.Sprintf("Test Place %d", i+1+offset)
		places[i] = map[string]string{
			"place_name":       placeName,
			"addr:city":        "City",
			"addr:street":      "Street",
			"addr:housenumber": fmt.Sprintf("%d", i+1+offset),
			"name":             placeName,
		}
	}
	return places
}

func BenchmarkProcessPlaces(b *testing.B) {
	// Инициализируем PostgreSQL
	database.InitDB()
	// Инициализируем Redis
	database.InitRedis()

	// Проверяем, что БД инициализирована
	db := database.GetDB()
	if db == nil {
		b.Fatal("database.GetDB() returned nil")
	}

	// Проверяем подключение к Redis
	ctx := context.Background()
	if _, err := database.RedisClient.Ping(ctx).Result(); err != nil {
		b.Fatal("Redis not connected: ", err)
	}

	// Создаем сервис
	service := services.NewPlaceService(db)

	userID := uint(1)
	places := generatePlaces(10, 0) // 50 мест, Test Place 1–50

	// Проверяем переменные окружения
	if os.Getenv("HOST_LLM") == "" || os.Getenv("HOST_TTS") == "" {
		b.Fatal("HOST_LLM or HOST_TTS not set")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.ProcessPlaces(userID, places)
		if err != nil {
			b.Fatalf("error in benchmark: %v", err)
		}
	}
}

func BenchmarkProcessPlacesGoroutines(b *testing.B) {
	// Инициализируем PostgreSQL
	database.InitDB()
	// Инициализируем Redis
	database.InitRedis()

	// Проверяем, что БД инициализирована
	db := database.GetDB()
	if db == nil {
		b.Fatal("database.GetDB() returned nil")
	}

	// Проверяем подключение к Redis
	ctx := context.Background()
	if _, err := database.RedisClient.Ping(ctx).Result(); err != nil {
		b.Fatal("Redis not connected: ", err)
	}

	// Создаем сервис
	service := services.NewPlaceService(db)

	userID := uint(1)
	places := generatePlaces(10, 10) // 50 мест, Test Place 51–100

	// Проверяем переменные окружения
	if os.Getenv("HOST_LLM") == "" || os.Getenv("HOST_TTS") == "" {
		b.Fatal("HOST_LLM or HOST_TTS not set")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := service.ProcessPlacesGoroutines(userID, places)
		if err != nil {
			b.Fatalf("error in benchmark: %v", err)
		}
	}
}
