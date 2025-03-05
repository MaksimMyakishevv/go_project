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

// PlaceService представляет сервис для работы с местами
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

// GetUserHistory возвращает историю запросов пользователя из PostgreSQL
func (s *PlaceService) GetUserHistory(userID uint) ([]models.Place, error) {
	var places []models.Place
	if err := s.DB.Where("user_id = ?", userID).Find(&places).Error; err != nil {
		return nil, err
	}
	return places, nil
}

// ProcessPlace отправляет данные о месте на обработку нейросетью с таймаутом 15 секунд
func (s *PlaceService) ProcessPlace(userID uint, placeData map[string]string) (string, bool, error) {
	ctx := context.Background()

	// Генерируем ключ для Redis с учетом userID и place_name
	placeName := placeData["place_name"]
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
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := &http.Client{}
	// Отправляем все поля: addr:city, addr:street, addr:housenumber, name
	reqBody := map[string]string{
		"addr:city":        placeData["addr:city"],
		"addr:street":      placeData["addr:street"],
		"addr:housenumber": placeData["addr:housenumber"],
		"name":             placeData["name"],
	}
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

	// Ищем разделитель
	separator := []byte("---AUDIO---")
	sepIndex := bytes.Index(body, separator)
	if sepIndex == -1 {
		return "", false, fmt.Errorf("разделитель ---AUDIO--- не найден")
	}

	// Извлекаем текст
	textBytes := body[:sepIndex]
	text := string(textBytes)

	// Извлекаем аудио (если нужно)
	audioData := body[sepIndex+len(separator):]
	if len(audioData) == 0 {
		fmt.Println("Отсутствует аудио")
	}

	// Сохраняем текст в Redis
	expiration := 24 * time.Hour
	if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
		fmt.Printf("Ошибка при сохранении данных в Redis: %v\n", err)
	}

	return text, false, nil
}

// ProcessPlaces обрабатывает массив мест и отправляет их на обработку нейросетью
func (s *PlaceService) ProcessPlaces(userID uint, places []map[string]string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	for _, place := range places {
		// Отправляем запрос на нейросеть с таймаутом 15 секунд
		result, isCached, err := s.ProcessPlace(userID, place)
		if err != nil {
			fmt.Println(err)
			// Если произошла ошибка, добавляем результат с ошибкой
			results = append(results, map[string]interface{}{
				"place_name": place["place_name"],
				"status":     "timeout or no connection",
				"response":   nil,
			})
			continue
		}

		// Если ответ НЕ из кеша, добавляем место в историю пользователя
		if !isCached {
			_, err = s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: place["place_name"]})
			if err != nil {
				// Если не удалось добавить место в историю, добавляем результат с ошибкой
				results = append(results, map[string]interface{}{
					"place_name": place["place_name"],
					"status":     "failed_to_add",
					"response":   nil,
				})
				continue
			}
		}

		// Сохраняем успешный результат
		results = append(results, map[string]interface{}{
			"place_name": place["place_name"],
			"status":     "success",
			"response":   result,
		})
	}

	return results, nil
}

// CleanupOldPlaces удаляет записи старше 1 часа
func (s *PlaceService) CleanupOldPlaces() {
	ticker := time.NewTicker(1 * time.Hour) // Запуск каждые 1 час
	defer ticker.Stop()

	for range ticker.C {
		cutoffTime := time.Now().Add(-1 * time.Hour)
		if err := s.DB.Where("created_at < ?", cutoffTime).Delete(&models.Place{}).Error; err != nil {
			fmt.Printf("Ошибка при удалении старых записей: %v\n", err)
		} else {
			fmt.Println("Старые записи успешно удалены")
		}
	}
}

// GetCachedResponse возвращает закешированный ответ из Redis
func (s *PlaceService) GetCachedResponse(userID uint, placeName string) (string, error) {
	ctx := context.Background()

	cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)
	cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		fmt.Printf("Ответ не найден в кеше для пользователя %d и места: %s\n", userID, placeName)
		return "", fmt.Errorf("ответ не найден в кеше для места: %s", placeName)
	} else if err != nil {
		fmt.Printf("Ошибка при получении данных из Redis: %v\n", err)
		return "", fmt.Errorf("ошибка при обращении к Redis: %v", err)
	}

	fmt.Printf("Ответ найден в кеше для пользователя %d и места: %s\n", userID, placeName)
	return cachedResponse, nil
}

// ProcessJSON обрабатывает JSON-файл и отправляет места на обработку
func (s *PlaceService) ProcessJSON(userID uint, osmObjects []dto.OSMObject) ([]map[string]interface{}, error) {
	var places []map[string]string

	for _, obj := range osmObjects {
		if obj.Type == "way" {
			//placeID := obj.ID
			tags := obj.Tags
			var placeName string

			// Проверяем наличие поля "name" в tags
			if name, exists := tags["name"]; exists && name != "" {
				placeName = name
			} else if street, exists := tags["addr:street"]; exists && street != "" {
				// Если "name" нет, формируем name из "addr:street" и "addr:housenumber"
				if housenumber, exists := tags["addr:housenumber"]; exists && housenumber != "" {
					placeName = fmt.Sprintf("%s, %s", street, housenumber)
				}
			}

			// Если удалось сформировать placeName, добавляем место с дополнительными полями
			if placeName != "" {
				placeData := map[string]string{
					"place_name":       placeName,
					"addr:city":        tags["addr:city"],
					"addr:street":      tags["addr:street"],
					"addr:housenumber": tags["addr:housenumber"],
					"name":             tags["name"],
				}
				places = append(places, placeData)
			}
		}
	}

	// Обрабатываем места с помощью существующего метода
	results, err := s.ProcessPlaces(userID, places)
	if err != nil {
		return nil, err
	}

	// Дополняем результаты place_id из исходных данных
	for i, result := range results {
		if status, ok := result["status"].(string); ok && status == "success" {
			result["place_id"] = osmObjects[i].ID
		}
	}

	return results, nil
}
