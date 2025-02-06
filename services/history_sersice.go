package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"new/database"
	"new/dto"
	"new/models"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type PlaceService struct {
	DB *gorm.DB
}

// NewPlaceService создает новый экземпляр PlaceService
func NewPlaceService(db *gorm.DB) *PlaceService {
	return &PlaceService{DB: db}
}

// AddPlace добавляет новое место в историю пользователя
func (s *PlaceService) AddPlace(userID uint, input dto.AddPlaceDTO) (*models.Place, error) {
	place := &models.Place{
		UserID:    userID,
		PlaceName: input.PlaceName,
	}

	if err := s.DB.Create(place).Error; err != nil {
		return nil, err
	}

	return place, nil
}

// GetUserHistory возвращает историю запросов пользователя иза postgress
func (s *PlaceService) GetUserHistory(userID uint) ([]models.Place, error) {
	var places []models.Place
	if err := s.DB.Where("user_id = ?", userID).Find(&places).Error; err != nil {
		return nil, err
	}
	return places, nil
}

// ProcessPlace отправляет место на обработку нейросети с таймаутом 5 секунд
func (s *PlaceService) ProcessPlace(userID uint, placeName string) (string, bool, error) {
	ctx := context.Background()

	// Генерируем ключ для Redis с учетом userID
	cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

	// Проверяем, есть ли ответ в Redis
	if cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
		fmt.Printf("Ответ найден в кеше для пользователя %d и места: %s\n", userID, placeName)
		return cachedResponse, true, nil // Возвращаем ответ и флаг "из кеша"
	} else if err != redis.Nil {
		// Если ошибка Redis не связана с отсутствием ключа, логируем её
		fmt.Printf("Ошибка при получении данных из Redis: %v\n", err)
	}

	// Если ответа нет в кеше, отправляем запрос к нейросети
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{}
	reqBody := map[string]string{"place_name": placeName}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctxWithTimeout, "POST", "http://127.0.0.1:8000/random_text", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", false, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	var neuralResponse struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &neuralResponse); err != nil {
		return "", false, err
	}

	// Сохраняем ответ в Redis
	expiration := 24 * time.Hour // Время жизни кеша (например, 24 часа)
	if err := database.RedisClient.Set(ctx, cacheKey, neuralResponse.Text, expiration).Err(); err != nil {
		fmt.Printf("Ошибка при сохранении данных в Redis: %v\n", err)
	}

	return neuralResponse.Text, false, nil // Возвращаем ответ и флаг "не из кеша"
}

// ProcessPlaces обрабатывает массив мест и отправляет их на обработку нейросетью
func (s *PlaceService) ProcessPlaces(userID uint, places []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	for _, placeName := range places {
		// Отправляем запрос на нейросеть с таймаутом 5 секунд
		result, isCached, err := s.ProcessPlace(userID, placeName)
		if err != nil {
			// Если произошла ошибка, добавляем результат с ошибкой
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "timeout or no conection", // или другая ошибка
				"response":   nil,
			})
			continue
		}

		// Если ответ НЕ из кеша, добавляем место в историю пользователя
		if !isCached {
			_, err = s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
			if err != nil {
				// Если не удалось добавить место в историю, добавляем результат с ошибкой
				results = append(results, map[string]interface{}{
					"place_name": placeName,
					"status":     "failed_to_add",
					"response":   nil,
				})
				continue
			}
		}

		// Сохраняем успешный результат
		results = append(results, map[string]interface{}{
			"place_name": placeName,
			"status":     "success",
			"response":   result,
		})
	}

	return results, nil
}

// CleanupOldPlaces удаляет записи старше 30 секунд
func (s *PlaceService) CleanupOldPlaces() {
	ticker := time.NewTicker(1 * time.Hour) // Запуск каждые 1 час
	defer ticker.Stop()

	for range ticker.C {
		cutoffTime := time.Now().Add(-1 * time.Hour) // Удаление записей старше оперделенного времени секунд
		if err := s.DB.Where("created_at < ?", cutoffTime).Delete(&models.Place{}).Error; err != nil {
			fmt.Printf("Ошибка при удалении старых записей: %v\n", err)
		} else {
			fmt.Println("Старые записи успешно удалены")
		}
	}
}

// GetCachedResponse возвращает закешированный ответ из Redis для конкретного пользователя и места
func (s *PlaceService) GetCachedResponse(userID uint, placeName string) (string, error) {
	ctx := context.Background()

	// Генерируем ключ для Redis с учетом userID и названия места
	cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

	// Проверяем, есть ли ответ в Redis
	cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		// Ключ не найден
		fmt.Printf("Ответ не найден в кеше для пользователя %d и места: %s\n", userID, placeName)
		return "", fmt.Errorf("ответ не найден в кеше для места: %s", placeName)
	} else if err != nil {
		// Ошибка Redis
		fmt.Printf("Ошибка при получении данных из Redis: %v\n", err)
		return "", fmt.Errorf("ошибка при обращении к Redis: %v", err)
	}

	// Логируем успешное получение ответа
	fmt.Printf("Ответ найден в кеше для пользователя %d и места: %s\n", userID, placeName)
	return cachedResponse, nil
}
