package database

import (
	"context"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client
var ctx = context.Background()

func InitRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost + ":" + redisPort, // Адрес Redis (например, "localhost:6379")
		Password: redisPassword,               // Пароль (если есть)
		DB:       0,                           // Номер базы данных Redis
	})

	// Проверяем подключение
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Ошибка подключения к Redis: %v", err)
	}

	log.Println("Подключение к Redis успешно установлено!")
}
