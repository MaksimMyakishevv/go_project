package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

// // ProcessPlace отправляет данные о месте на обработку нейросетью с таймаутом 15 секунд
// func (s *PlaceService) ProcessPlace(userID uint, placeData map[string]string) (string, bool, []byte, error) {
// 	ctx := context.Background()

// 	// Генерируем ключ для Redis с учетом userID и place_name
// 	placeName := placeData["place_name"]
// 	cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

// 	// Проверяем, есть ли ответ в Redis
// 	if cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
// 		fmt.Printf("Ответ найден в кеше для пользователя %d и места: %s\n", userID, placeName)
// 		// Если данные есть в кеше, возвращаем только текст (аудио не генерируем)
// 		return cachedResponse, true, nil, nil
// 	} else if err != redis.Nil {
// 		// Если ошибка Redis не связана с отсутствием ключа, логируем её
// 		fmt.Printf("Ошибка при получении данных из Redis: %v\n", err)
// 		return "", false, nil, fmt.Errorf("ошибка Redis: %v", err)
// 	}

// 	// Если ответа нет в кеше, отправляем запрос к LLM
// 	text, err := s.SendToLLM(placeData)
// 	if err != nil {
// 		return "", false, nil, fmt.Errorf("ошибка при отправке данных в LLM: %v", err)
// 	}

// 	// Генерируем аудио
// 	audioData, err := s.AudioGenerate(text)
// 	if err != nil {
// 		return "", false, nil, fmt.Errorf("ошибка при генерации аудио: %v", err)
// 	}

// 	// Сохраняем текст в Redis
// 	expiration := 24 * time.Hour
// 	if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
// 		fmt.Printf("Ошибка при сохранении данных в Redis: %v\n", err)
// 	}

// 	// Возвращаем текст, флаг "из кеша", аудио и nil (ошибки нет)
// 	return text, false, audioData, nil
// }

// SendBatchToLLM отправляет данные одного места в LLM в формате одиночного объекта JSON
func (s *PlaceService) SendBatchToLLM(places []map[string]string) (string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Поскольку мы работаем с заглушкой, берём только первый элемент
	if len(places) == 0 {
		return "", fmt.Errorf("массив мест пуст")
	}
	place := places[0]

	// Формируем JSON вручную в формате одиночного объекта
	var jsonBuilder strings.Builder
	jsonBuilder.WriteString("{")

	// Поля в точном порядке, как в PlaceData
	fields := []struct {
		key   string
		value string
	}{
		{"addr:city", place["addr:city"]},
		{"addr:street", place["addr:street"]},
		{"addr:housenumber", place["addr:housenumber"]},
		{"name", place["name"]},
		{"amenity", place["amenity"]},
		{"tourism", place["tourism"]},
		{"highway", place["highway"]},
		{"leisure", place["leisure"]},
		{"building", place["building"]},
		{"inscription", place["inscription"]},
		{"description", place["description"]},
	}

	for i, field := range fields {
		if i > 0 {
			jsonBuilder.WriteString(",")
		}
		// Экранируем значение для JSON
		escapedValue := strings.ReplaceAll(field.value, `"`, `\"`)
		jsonBuilder.WriteString(fmt.Sprintf(`"%s":"%s"`, field.key, escapedValue))
	}

	jsonBuilder.WriteString("}")
	jsonBody := jsonBuilder.String()

	req, err := http.NewRequestWithContext(ctxWithTimeout, "POST", os.Getenv("HOST_LLM"), bytes.NewBufferString(jsonBody))
	if err != nil {
		return "", fmt.Errorf("ошибка при создании запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ошибка: статус ответа %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка при чтении тела ответа: %v", err)
	}

	var response struct {
		Text string `json:"message"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	// Проверяем текст на ошибки
	if err := checkResponseError(response.Text, "LLM"); err != nil {
		return "", err
	}

	return response.Text, nil
}

// // SendBatchToLLM отправляет массив мест в LLM одним запросом и возвращает единый текст
// func (s *PlaceService) SendBatchToLLM(places []map[string]string) (string, error) {
// 	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	jsonBody, err := json.Marshal(places)
// 	if err != nil {
// 		return "", fmt.Errorf("ошибка при маршалинге JSON: %v", err)
// 	}

// 	req, err := http.NewRequestWithContext(ctxWithTimeout, "POST", os.Getenv("HOST_LLM"), bytes.NewBuffer(jsonBody))
// 	if err != nil {
// 		return "", fmt.Errorf("ошибка при создании запроса: %v", err)
// 	}
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("ошибка при отправке запроса: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return "", fmt.Errorf("ошибка: статус ответа %d", resp.StatusCode)
// 	}

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", fmt.Errorf("ошибка при чтении тела ответа: %v", err)
// 	}

// 	var response struct {
// 		Text string `json:"message"` // Предполагаем, что LLM возвращает текст в поле "message"
// 	}
// 	if err := json.Unmarshal(body, &response); err != nil {
// 		return "", fmt.Errorf("ошибка парсинга JSON: %v", err)
// 	}
// 	// Проверяем текст на ошибки
// 	if err := checkResponseError(response.Text, "LLM"); err != nil {
// 		return "", err
// 	}

// 	return response.Text, nil
// }

// AudioGenerate отправляет текст в формате JSON и получает аудио в кодировке UTF-8
func (s *PlaceService) AudioGenerate(text string) ([]byte, error) {
	// Создаем HTTP-клиент
	client := &http.Client{}

	// Формируем тело запроса в формате JSON
	reqBody := map[string]string{"message": text}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка при маршалинге JSON: %v", err)
	}

	// Создаем контекст с тайм-аутом
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Создаем HTTP-запрос
	req, err := http.NewRequestWithContext(ctx, "POST", os.Getenv("HOST_TTS"), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании запроса: %v", err)
	}

	// Устанавливаем заголовок Content-Type
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка: статус ответа %d", resp.StatusCode)
	}

	// Читаем тело ответа (аудио в кодировке UTF-8)
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении тела ответа: %v", err)
	}

	// Проверяем, является ли ответ текстом ошибки
	var response struct {
		Text string `json:"message"`
	}
	if err := json.Unmarshal(audioData, &response); err == nil {
		// Если удалось распарсить JSON и есть поле "message", проверяем на ошибку
		if err := checkResponseError(response.Text, "TTS"); err != nil {
			return nil, err
		}
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("пустое тело ответа")
	}

	// Возвращаем аудио
	return audioData, nil
}

// checkResponseError проверяет текст ответа на наличие ошибок и возвращает error, если ошибка найдена
func checkResponseError(responseText string, source string) error {
	if strings.Contains(responseText, "ОШИБКА") {
		return fmt.Errorf("ошибка от %s: %s", source, responseText)
	}
	return nil
}

// ProcessPlaces обрабатывает массив мест батч-обработкой (заглушка: только первый элемент)
func (s *PlaceService) ProcessPlaces(userID uint, places []map[string]string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	ctx := context.Background()

	// Заглушка: если массив пустой, возвращаем пустой результат
	if len(places) == 0 {
		return results, nil
	}

	// Берем только первый элемент массива
	place := places[0]
	placeName := place["place_name"]
	cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

	// Проверяем кеш для первого места
	if cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
		results = append(results, map[string]interface{}{
			"place_name": placeName,
			"status":     "success",
			"response":   cachedResponse,
			"audio":      nil,
		})
		return results, nil
	}

	// Формируем массив с одним элементом для отправки в LLM
	placesToProcess := []map[string]string{place}

	// Отправляем запрос в LLM
	text, err := s.SendBatchToLLM(placesToProcess)
	if err != nil {
		results = append(results, map[string]interface{}{
			"place_name": placeName,
			"status":     "llm_error",
			"response":   err.Error(),
			"audio":      nil,
		})
		return results, err
	}

	// Генерируем аудио
	audioData, err := s.AudioGenerate(text)
	if err != nil {
		results = append(results, map[string]interface{}{
			"place_name": placeName,
			"status":     "tts_error",
			"response":   text,
			"audio":      nil,
		})
		return results, err
	}

	// Сохраняем результат в Redis
	expiration := 24 * time.Hour
	if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
		fmt.Printf("Ошибка при сохранении в Redis: %v\n", err)
	}

	// Добавляем в историю
	_, err = s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
	if err != nil {
		results = append(results, map[string]interface{}{
			"place_name": placeName,
			"status":     "failed_to_add",
			"response":   text,
			"audio":      audioData,
		})
		return results, err
	}

	// Успешный результат
	results = append(results, map[string]interface{}{
		"place_name": placeName,
		"status":     "success",
		"response":   text,
		"audio":      audioData,
	})

	return results, nil
}

// ProcessPlaces обрабатывает массив мест батч-обработкой (тоесть полный пул обработанных данных)
// func (s *PlaceService) ProcessPlaces(userID uint, places []map[string]string) ([]map[string]interface{}, error) {
// 	var results []map[string]interface{}
// 	ctx := context.Background()

// 	placeTexts := make(map[string]string)
// 	for _, place := range places {
// 		placeName := place["place_name"]
// 		cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

// 		if cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
// 			placeTexts[placeName] = cachedResponse
// 			results = append(results, map[string]interface{}{
// 				"place_name": placeName,
// 				"status":     "success",
// 				"response":   cachedResponse,
// 				"audio":      nil,
// 			})
// 		}
// 	}

// 	var placesToProcess []map[string]string
// 	for _, place := range places {
// 		placeName := place["place_name"]
// 		if _, exists := placeTexts[placeName]; !exists {
// 			placesToProcess = append(placesToProcess, place)
// 		}
// 	}

// 	if len(placesToProcess) > 0 {
// 		text, err := s.SendBatchToLLM(placesToProcess)
// 		if err != nil {
// 			for _, place := range placesToProcess {
// 				results = append(results, map[string]interface{}{
// 					"place_name": place["place_name"],
// 					"status":     "llm_error",
// 					"response":   err.Error(),
// 					"audio":      nil,
// 				})
// 			}
// 			return results, err
// 		}

// 		audioData, err := s.AudioGenerate(text)
// 		if err != nil {
// 			for _, place := range placesToProcess {
// 				results = append(results, map[string]interface{}{
// 					"place_name": place["place_name"],
// 					"status":     "tts_error", // Изменяем статус для ошибок TTS
// 					"response":   text,
// 					"audio":      nil,
// 				})
// 			}
// 			return results, err
// 		}

// 		for _, place := range placesToProcess {
// 			placeName := place["place_name"]

// 			cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)
// 			expiration := 24 * time.Hour
// 			if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
// 				fmt.Printf("Ошибка при сохранении в Redis: %v\n", err)
// 			}

// 			_, err = s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
// 			if err != nil {
// 				results = append(results, map[string]interface{}{
// 					"place_name": placeName,
// 					"status":     "failed_to_add",
// 					"response":   text,
// 					"audio":      audioData,
// 				})
// 				continue
// 			}

// 			results = append(results, map[string]interface{}{
// 				"place_name": placeName,
// 				"status":     "success",
// 				"response":   text,
// 				"audio":      audioData,
// 			})
// 		}
// 	}

// 	return results, nil
// }

// ProcessJSON обрабатывает JSON-файл и отправляет места на обработку
func (s *PlaceService) ProcessJSON(userID uint, osmObjects []dto.OSMObject) ([]map[string]interface{}, error) {
	var places []map[string]string

	for _, obj := range osmObjects {
		tags := obj.Tags
		var placeName string

		// 1. Формируем placeName
		if name, exists := tags["name"]; exists && name != "" {
			placeName = name
		} else if street, exists := tags["addr:street"]; exists && street != "" {
			if housenumber, exists := tags["addr:housenumber"]; exists && housenumber != "" {
				placeName = fmt.Sprintf("%s, %s", street, housenumber)
			} else {
				placeName = street // Если номера дома нет, используем только улицу
			}
		} else {
			// 2. Дополнительные варианты для placeName
			switch {
			case tags["inscription"] != "":
				placeName = tags["inscription"]
			case tags["description"] != "":
				if len(tags["description"]) > 100 { // Ограничиваем длину
					placeName = tags["description"][:100] + "..."
				} else {
					placeName = tags["description"]
				}
			case tags["amenity"] != "":
				placeName = fmt.Sprintf("%s %d", tags["amenity"], obj.ID)
			case tags["tourism"] != "":
				placeName = fmt.Sprintf("%s %d", tags["tourism"], obj.ID)
			case tags["highway"] != "":
				placeName = fmt.Sprintf("%s %d", tags["highway"], obj.ID)
			case tags["leisure"] != "":
				placeName = fmt.Sprintf("%s %d", tags["leisure"], obj.ID)
			case tags["building"] != "":
				placeName = fmt.Sprintf("building %d", obj.ID)
			default:
				placeName = fmt.Sprintf("%s %d", obj.Type, obj.ID) // Fallback
			}
		}

		// 3. Собираем данные о месте
		placeData := map[string]string{
			"type":             obj.Type,
			"place_name":       placeName,
			"addr:city":        tags["addr:city"],
			"addr:street":      tags["addr:street"],
			"addr:housenumber": tags["addr:housenumber"],
			"name":             tags["name"],
			"amenity":          tags["amenity"],
			"tourism":          tags["tourism"],
			"highway":          tags["highway"],
			"leisure":          tags["leisure"],
			"building":         tags["building"],
			"inscription":      tags["inscription"],
			"description":      tags["description"],
		}

		// 4. Добавляем координаты для node
		if obj.Type == "node" {
			placeData["lat"] = fmt.Sprintf("%f", obj.Lat)
			placeData["lon"] = fmt.Sprintf("%f", obj.Lon)
		}

		places = append(places, placeData)
	}

	// 5. Передаем в ProcessPlaces (предполагаем, что он возвращает результаты)
	results, err := s.ProcessPlaces(userID, places)
	if err != nil {
		return nil, err
	}

	// 6. Добавляем place_id в успешные результаты
	for i, result := range results {
		if status, ok := result["status"].(string); ok && status == "success" {
			result["place_id"] = fmt.Sprintf("%d", osmObjects[i].ID)
		}
	}

	return results, nil
}

//
//
/*		НИЖЕ ВЕРСИЯ БЕЗ АУТЕНТИФИКАЦИИ      */
//
//


func (s *PlaceService) ProcessJSONNoAuth(osmObjects []dto.OSMObject) ([]map[string]interface{}, error) {
	var places []map[string]string

	for _, obj := range osmObjects {
		tags := obj.Tags
		var placeName string

		// 1. Формируем placeName
		if name, exists := tags["name"]; exists && name != "" {
			placeName = name
		} else if street, exists := tags["addr:street"]; exists && street != "" {
			if housenumber, exists := tags["addr:housenumber"]; exists && housenumber != "" {
				placeName = fmt.Sprintf("%s, %s", street, housenumber)
			} else {
				placeName = street // Если номера дома нет, используем только улицу
			}
		} else {
			// 2. Дополнительные варианты для placeName
			switch {
			case tags["inscription"] != "":
				placeName = tags["inscription"]
			case tags["description"] != "":
				if len(tags["description"]) > 100 { // Ограничиваем длину
					placeName = tags["description"][:100] + "..."
				} else {
					placeName = tags["description"]
				}
			case tags["amenity"] != "":
				placeName = fmt.Sprintf("%s %d", tags["amenity"], obj.ID)
			case tags["tourism"] != "":
				placeName = fmt.Sprintf("%s %d", tags["tourism"], obj.ID)
			case tags["highway"] != "":
				placeName = fmt.Sprintf("%s %d", tags["highway"], obj.ID)
			case tags["leisure"] != "":
				placeName = fmt.Sprintf("%s %d", tags["leisure"], obj.ID)
			case tags["building"] != "":
				placeName = fmt.Sprintf("building %d", obj.ID)
			default:
				placeName = fmt.Sprintf("%s %d", obj.Type, obj.ID) // Fallback
			}
		}

		// 3. Собираем данные о месте
		placeData := map[string]string{
			"type":             obj.Type,
			"place_name":       placeName,
			"addr:city":        tags["addr:city"],
			"addr:street":      tags["addr:street"],
			"addr:housenumber": tags["addr:housenumber"],
			"name":             tags["name"],
			"amenity":          tags["amenity"],
			"tourism":          tags["tourism"],
			"highway":          tags["highway"],
			"leisure":          tags["leisure"],
			"building":         tags["building"],
			"inscription":      tags["inscription"],
			"description":      tags["description"],
		}

		// 4. Добавляем координаты для node
		if obj.Type == "node" {
			placeData["lat"] = fmt.Sprintf("%f", obj.Lat)
			placeData["lon"] = fmt.Sprintf("%f", obj.Lon)
		}

		places = append(places, placeData)
	}

	// 5. Передаем в ProcessPlaces (предполагаем, что он возвращает результаты)
	results, err := s.ProcessPlacesNoAuth(places)
	if err != nil {
		return nil, err
	}

	// 6. Добавляем place_id в успешные результаты
	for i, result := range results {
		if status, ok := result["status"].(string); ok && status == "success" {
			result["place_id"] = fmt.Sprintf("%d", osmObjects[i].ID)
		}
	}

	return results, nil
}

// func (s *PlaceService) ProcessPlacesNoAuth0(places []map[string]string) ([]map[string]interface{}, error) {
// 	var results []map[string]interface{}

// 	for _, place := range places {
// 		// Отправляем запрос на нейросеть с таймаутом 15 секунд
// 		text, audioData, err := s.ProcessPlaceNoAuth(place)
// 		if err != nil {
// 			fmt.Println(err)
// 			// Если произошла ошибка, добавляем результат с ошибкой
// 			results = append(results, map[string]interface{}{
// 				"place_name": place["place_name"],
// 				"status":     "timeout or no connection",
// 				"response":   nil,
// 				"audio":      nil,
// 			})
// 			continue
// 		}

// 		// Сохраняем успешный результат
// 		results = append(results, map[string]interface{}{
// 			"place_name": place["place_name"],
// 			"status":     "success",
// 			"response":   text,      // Текстовый ответ
// 			"audio":      audioData, // Аудио в кодировке UTF-8
// 		})
// 	}

// 	return results, nil
// }

// func (s *PlaceService) ProcessPlaceNoAuth(placeData map[string]string) (string, []byte, error) {

// 	// Если ответа нет в кеше, отправляем запрос к LLM
// 	text, err := s.SendToLLM(placeData)
// 	if err != nil {
// 		return "", nil, fmt.Errorf("ошибка при отправке данных в LLM: %v", err)
// 	}

// 	// Генерируем аудио
// 	audioData, err := s.AudioGenerate(text)
// 	if err != nil {
// 		return "", nil, fmt.Errorf("ошибка при генерации аудио: %v", err)
// 	}

// 	// Возвращаем текст, флаг "из кеша", аудио и nil (ошибки нет)
// 	return text, audioData, nil
// }

func (s *PlaceService) ProcessPlacesNoAuth(places []map[string]string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	//ctx := context.Background()

	// Заглушка: если массив пустой, возвращаем пустой результат
	if len(places) == 0 {
		return results, nil
	}

	for _, place := range places {

		placeName := place["place_name"]

		// Формируем массив с одним элементом для отправки в LLM
		placesToProcess := []map[string]string{place}

		// Отправляем запрос в LLM
		text, err := s.SendBatchToLLM(placesToProcess)
		if err != nil {
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "llm_error",
				"response":   err.Error(),
				"audio":      nil,
			})
			return results, nil //err
		}

		// Генерируем аудио
		audioData, err := s.AudioGenerate(text)
		if err != nil {
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "tts_error",
				"response":   text,
				"audio":      nil,
			})
			return results, nil //err
		}

		// Успешный результат
		results = append(results, map[string]interface{}{
			"place_name": placeName,
			"status":     "success",
			"response":   text,
			"audio":      audioData,
		})
	}

	return results, nil
}