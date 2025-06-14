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
	"sync"
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

// SendBatchToLLM отправляет данные одного места в LLM в формате одиночного объекта JSON
func (s *PlaceService) SendBatchToLLM(places []map[string]string) (string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 9999*time.Second)
	defer cancel()

	if len(places) == 0 {
		return "", fmt.Errorf("массив мест пуст")
	}

	// Берем только первое место, так как ожидается один объект
	place := places[0]

	// Формируем JSON вручную с помощью strings.Builder
	var jsonBuilder strings.Builder
	jsonBuilder.WriteString("{") // Начинаем с объекта

	// Поля для места
	fields := []string{
		"addr:city",
		"addr:street",
		"addr:housenumber",
		"name",
		"amenity",
		"tourism",
		"highway",
		"leisure",
		"building",
		"inscription",
		"description",
	}

	firstField := true // Флаг для отслеживания первого добавленного поля
	for _, field := range fields {
		if value, exists := place[field]; exists && value != "" {
			if !firstField {
				jsonBuilder.WriteString(",")
			}
			escapedValue := strings.ReplaceAll(value, `"`, `\"`)
			jsonBuilder.WriteString(fmt.Sprintf(`"%s":"%s"`, field, escapedValue))
			firstField = false // После добавления первого поля сбрасываем флаг
		}
	}

	jsonBuilder.WriteString("}") // Закрываем объект
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

	if err := checkResponseError(response.Text, "LLM"); err != nil {
		return "", err
	}

	return response.Text, nil
}

// AudioGenerate отправляет текст в формате JSON и получает аудио в кодировке UTF-8
func (s *PlaceService) AudioGenerate(text string) ([]byte, error) {
	client := &http.Client{}
	reqBody := map[string]string{"message": text}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка при маршалинге JSON: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 9999*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", os.Getenv("HOST_TTS"), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка: статус ответа %d", resp.StatusCode)
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении тела ответа: %v", err)
	}

	var response struct {
		Text string `json:"message"`
	}
	if err := json.Unmarshal(audioData, &response); err == nil {
		if err := checkResponseError(response.Text, "TTS"); err != nil {
			return nil, err
		}
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("пустое тело ответа")
	}

	return audioData, nil
}

// checkResponseError проверяет текст ответа на наличие ошибок
func checkResponseError(responseText string, source string) error {
	if strings.Contains(responseText, "ОШИБКА") {
		return fmt.Errorf("ошибка от %s: %s", source, responseText)
	}
	return nil
}

// ProcessPlaces обрабатывает массив мест последовательно, возвращая массив с полной информацией о каждом месте
func (s *PlaceService) ProcessPlaces(userID uint, places []map[string]string) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	ctx := context.Background()

	if len(places) == 0 {
		return result, nil
	}

	// Обрабатываем каждое место по очереди
	for _, place := range places {
		placeName := place["place_name"]
		cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

		// Инициализируем объект для текущего места
		placeResult := map[string]interface{}{
			"place_name": placeName,
			"details":    place, // Сохраняем исходные данные места
			"status":     "pending",
		}

		// Проверяем кеш
		cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == redis.Nil {
			// Если нет в кеше, отправляем в LLM индивидуально
			placeToProcess := []map[string]string{place} // Массив из одного объекта
			text, llmErr := s.SendBatchToLLM(placeToProcess)
			if llmErr != nil {
				placeResult["response"] = fmt.Sprintf("Ошибка LLM: %v", llmErr)
				placeResult["audio"] = nil
				placeResult["status"] = "llm_error"
				result = append(result, placeResult)
				continue
			}

			// Генерируем аудио для этого конкретного ответа
			audioData, ttsErr := s.AudioGenerate(text)
			if ttsErr != nil {
				placeResult["response"] = text
				placeResult["audio"] = fmt.Sprintf("Ошибка TTS: %v", ttsErr)
				placeResult["status"] = "tts_error"
				result = append(result, placeResult)
				continue
			}

			// Сохраняем в кеш
			expiration := 24 * time.Hour
			if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
				fmt.Printf("Ошибка при сохранении в Redis: %v\n", err)
			}

			// Добавляем в историю
			_, err = s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
			if err != nil {
				fmt.Printf("Ошибка при добавлении в историю: %v\n", err)
				placeResult["response"] = text
				placeResult["audio"] = audioData
				placeResult["status"] = "failed_to_add"
				result = append(result, placeResult)
				continue
			}

			// Успешный результат
			placeResult["response"] = text
			placeResult["audio"] = audioData
			placeResult["status"] = "success"
		} else if err != nil {
			fmt.Printf("Ошибка при получении из Redis: %v\n", err)
			placeResult["response"] = fmt.Sprintf("Ошибка Redis: %v", err)
			placeResult["audio"] = nil
			placeResult["status"] = "cache_error"
		} else {
			// Если найдено в кеше
			placeResult["response"] = cachedResponse
			placeResult["audio"] = nil // Аудио не кэшируется
			placeResult["status"] = "success"
		}

		result = append(result, placeResult)
	}

	return result, nil
}

// ProcessPlacesGoroutines обрабатывает массив мест параллельно с оптимизацией
func (s *PlaceService) ProcessPlacesGoroutines(userID uint, places []map[string]string, resultChan chan<- map[string]interface{}) {
	ctx := context.Background()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 1) // Ограничиваем до 5 параллельных запросов

	if len(places) == 0 {
		return
	}

	// Обрабатываем каждое место
	for _, place := range places {
		placeName := place["place_name"]
		cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

		// Инициализируем объект результата
		placeResult := map[string]interface{}{
			"place_name": placeName,
			"details":    place,
			"status":     "pending",
		}

		// Проверяем кеш заранее
		if cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
			placeResult["response"] = cachedResponse
			audioData, _ := s.AudioGenerate(cachedResponse)
			placeResult["audio"] = audioData
			placeResult["status"] = "success"
			resultChan <- placeResult
			continue
		}

		// Если нет в кеше, обрабатываем в горутине
		wg.Add(1)
		semaphore <- struct{}{} // Захватываем слот в семафоре
		go func(place map[string]string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Освобождаем слот

			placeName := place["place_name"]
			cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)
			placeResult := map[string]interface{}{
				"place_name": placeName,
				"details":    place,
				"status":     "pending",
			}

			placeToProcess := []map[string]string{place}
			text, llmErr := s.SendBatchToLLM(placeToProcess)
			if llmErr != nil {
				placeResult["response"] = fmt.Sprintf("Ошибка LLM: %v", llmErr)
				placeResult["audio"] = nil
				placeResult["status"] = "llm_error"
				resultChan <- placeResult
				return
			}

			audioData, ttsErr := s.AudioGenerate(text)
			if ttsErr != nil {
				placeResult["response"] = text
				placeResult["audio"] = fmt.Sprintf("Ошибка TTS: %v", ttsErr)
				placeResult["status"] = "tts_error"
				resultChan <- placeResult
				return
			}

			expiration := 24 * time.Hour
			if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
				fmt.Printf("Ошибка при сохранении в Redis: %v\n", err)
			}

			_, err := s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
			if err != nil {
				fmt.Printf("Ошибка при добавлении в историю: %v\n", err)
				placeResult["response"] = text
				placeResult["audio"] = audioData
				placeResult["status"] = "failed_to_add"
				resultChan <- placeResult
				return
			}

			placeResult["response"] = text
			placeResult["audio"] = audioData
			placeResult["status"] = "success"
			resultChan <- placeResult
		}(place)
	}

	wg.Wait()
}

// findResultByPlaceName проверяет, есть ли результат для placeName
func findResultByPlaceName(results []map[string]interface{}, placeName string) (map[string]interface{}, bool) {
	for _, result := range results {
		if result["place_name"] == placeName {
			return result, true
		}
	}
	return nil, false
}

// ProcessJSON обрабатывает JSON-файл и отправляет места на обработку
// func (s *PlaceService) ProcessJSON(userID uint, osmObjects []dto.OSMObject) ([]map[string]interface{}, error) {
// 	places := make([]map[string]string, 0)

// 	// Формируем список мест
// 	for _, obj := range osmObjects {
// 		// Пропускаем пустые места
// 		if isPlaceEmpty(obj.Tags) {
// 			continue
// 		}

// 		// Формируем данные для ProcessPlaces
// 		placeData := buildPlaceData(obj)
// 		places = append(places, placeData)
// 	}

// 	// Обрабатываем места
// 	results, err := s.ProcessPlacesGoroutines(userID, places)
// 	if err != nil {
// 		return results, err
// 	}

// 	// Обновляем результаты, добавляя полную информацию о местах
// 	for i, result := range results {
// 		place := osmObjects[i]
// 		placeOutput := buildPlaceOutput(place)
// 		// Копируем все поля из placeOutput в result
// 		for k, v := range placeOutput {
// 			result[k] = v
// 		}
// 		// Удаляем временное поле details, если оно больше не нужно
// 		delete(result, "details")
// 	}

// 	return results, nil
// }

// buildPlaceName формирует название места
func buildPlaceName(tags map[string]string, objType string, objID int64) string {
	if name, exists := tags["name"]; exists && name != "" {
		return name
	}
	return fmt.Sprintf("%s %d", objType, objID)
}

// isPlaceEmpty проверяет, является ли место пустым
func isPlaceEmpty(tags map[string]string) bool {
	return tags["addr:street"] == "" &&
		tags["addr:housenumber"] == "" &&
		tags["addr:city"] == "" &&
		tags["name"] == ""
}

// buildPlaceData формирует данные для ProcessPlaces
func buildPlaceData(obj dto.OSMObject) map[string]string {
	tags := obj.Tags
	placeData := map[string]string{
		"type":             obj.Type,
		"place_name":       buildPlaceName(tags, obj.Type, obj.ID),
		"addr:city":        tags["addr:city"],
		"addr:street":      tags["addr:street"],
		"addr:housenumber": tags["addr:housenumber"],
		"name":             tags["name"],
	}
	if obj.Lat != 0 && obj.Lon != 0 {
		placeData["lat"] = fmt.Sprintf("%f", obj.Lat)
		placeData["lon"] = fmt.Sprintf("%f", obj.Lon)
	}
	return placeData
}

// buildPlaceOutput формирует выходное представление места
func buildPlaceOutput(obj dto.OSMObject) map[string]interface{} {
	tags := obj.Tags
	placeData := map[string]interface{}{
		"place_name": buildPlaceName(tags, obj.Type, obj.ID),
		"address":    "",
		"city":       tags["addr:city"],
		"place_id":   fmt.Sprintf("%d", obj.ID),
	}

	// Формируем address
	var addressParts []string
	if street := tags["addr:street"]; street != "" {
		addressParts = append(addressParts, street)
	}
	if housenumber := tags["addr:housenumber"]; housenumber != "" {
		addressParts = append(addressParts, housenumber)
	}
	placeData["address"] = strings.Join(addressParts, ", ")

	// Добавляем nodes
	if obj.Nodes != nil && len(obj.Nodes) > 0 {
		placeData["nodes"] = obj.Nodes
	}

	// Добавляем координаты
	if obj.Lat != 0 && obj.Lon != 0 {
		placeData["lat"] = obj.Lat
		placeData["lon"] = obj.Lon
	}

	return placeData
}

// StreamProcessJSON обрабатывает JSON-файл и отправляет результаты в канал по мере готовности
func (s *PlaceService) StreamProcessJSON(userID uint, osmObjects []dto.OSMObject, resultChan chan<- map[string]interface{}) {
	places := make([]map[string]string, 0)

	// Формируем список мест
	for _, obj := range osmObjects {
		// Пропускаем пустые места
		if isPlaceEmpty(obj.Tags) {
			continue
		}

		// Формируем данные для ProcessPlaces
		placeData := buildPlaceData(obj)
		places = append(places, placeData)
	}

	// Обрабатываем места и отправляем результаты в канал
	s.ProcessPlacesGoroutines(userID, places, resultChan)
}

//
//
/*		НИЖЕ ВЕРСИЯ БЕЗ АУТЕНТИФИКАЦИИ      */
//
//

// ProcessJSONNoAuth обрабатывает JSON-файл и отправляет места на обработку без аутентификации
func (s *PlaceService) ProcessJSONNoAuth(osmObjects []dto.OSMObject) ([]map[string]interface{}, error) {
	places := make([]map[string]string, 0)

	// Формируем список мест
	for _, obj := range osmObjects {
		// Пропускаем пустые места
		if isPlaceEmpty(obj.Tags) {
			continue
		}

		// Формируем данные для ProcessPlaces
		placeData := buildPlaceData(obj)
		places = append(places, placeData)
	}

	// Обрабатываем места
	results, err := s.ProcessPlacesNoAuth(places)
	if err != nil {
		return results, err
	}

	// Обновляем результаты, добавляя полную информацию о местах
	for i, result := range results {
		place := osmObjects[i]
		placeOutput := buildPlaceOutput(place)
		// Копируем все поля из placeOutput в result
		for k, v := range placeOutput {
			result[k] = v
		}
		// Удаляем временное поле details, если оно больше не нужно
		delete(result, "details")
		// Добавляем place_id в успешные результаты
		if status, ok := result["status"].(string); ok && status == "success" {
			result["place_id"] = fmt.Sprintf("%d", osmObjects[i].ID)
		}
	}

	return results, nil
}

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
			continue
			//return results, nil //err
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
			continue
			//return results, nil //err
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

//  MISTRAL ниже

// ProcessJSONMistral обрабатывает JSON-файл и отправляет места на обработку для Mistral
func (s *PlaceService) ProcessJSONMistral(osmObjects []dto.OSMObject) ([]map[string]interface{}, error) {
	places := make([]map[string]string, 0)

	// Формируем список мест
	for _, obj := range osmObjects {
		// Пропускаем пустые места
		if isPlaceEmpty(obj.Tags) {
			continue
		}

		// Формируем данные для ProcessPlaces
		placeData := buildPlaceData(obj)
		places = append(places, placeData)
	}

	// Обрабатываем места
	results, err := s.ProcessPlacesMistral(places)
	if err != nil {
		return results, err
	}

	// Обновляем результаты, добавляя полную информацию о местах
	for i, result := range results {
		place := osmObjects[i]
		placeOutput := buildPlaceOutput(place)
		// Копируем все поля из placeOutput в result
		for k, v := range placeOutput {
			result[k] = v
		}
		// Удаляем временное поле details, если оно больше не нужно
		delete(result, "details")
		// Добавляем place_id в успешные результаты
		if status, ok := result["status"].(string); ok && status == "success" {
			result["place_id"] = fmt.Sprintf("%d", osmObjects[i].ID)
		}
	}

	return results, nil
}

func (s *PlaceService) ProcessPlacesMistral(places []map[string]string) ([]map[string]interface{}, error) {
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
		text, err := s.SendBatchToMistral(placesToProcess)
		if err != nil {
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "llm_error",
				"response":   err.Error(),
				"audio":      nil,
			})
			continue
			//return results, nil //err
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
			continue
			//return results, nil //err
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

func (s *PlaceService) SendBatchToMistral(places []map[string]string) (string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	if len(places) == 0 {
		return "", fmt.Errorf("массив мест пуст")
	}

	// Берем только первое место, так как ожидается один объект
	place := places[0]

	// Формируем JSON вручную с помощью strings.Builder
	var jsonBuilder strings.Builder
	jsonBuilder.WriteString("{") // Начинаем с объекта

	// Поля для места
	fields := []string{
		"addr:city",
		"addr:street",
		"addr:housenumber",
		"name",
		"amenity",
		"tourism",
		"highway",
		"leisure",
		"building",
		"inscription",
		"description",
		"lat",
		"lon",
	}

	firstField := true // Флаг для отслеживания первого добавленного поля
	for _, field := range fields {
		if value, exists := place[field]; exists && value != "" {
			if !firstField {
				jsonBuilder.WriteString(",")
			}
			escapedValue := strings.ReplaceAll(value, `"`, `\"`)
			jsonBuilder.WriteString(fmt.Sprintf(`"%s":"%s"`, field, escapedValue))
			firstField = false // После добавления первого поля сбрасываем флаг
		}
	}

	jsonBuilder.WriteString("}") // Закрываем объект
	jsonBody := jsonBuilder.String()

	req, err := http.NewRequestWithContext(ctxWithTimeout, "POST", os.Getenv("HOST_MISTRAL"), bytes.NewBufferString(jsonBody))
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

	if err := checkResponseError(response.Text, "LLM"); err != nil {
		return "", err
	}

	return response.Text, nil
}

// func (s *PlaceService) SendBatchToMistral(places []map[string]string) (string, error) {
// 	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	// Поскольку мы работаем с заглушкой, берём только первый элемент
// 	if len(places) == 0 {
// 		return "", fmt.Errorf("массив мест пуст")
// 	}
// 	place := places[0]

// 	// Формируем JSON вручную в формате одиночного объекта
// 	var jsonBuilder strings.Builder
// 	jsonBuilder.WriteString("{")

// 	// Поля в точном порядке, как в PlaceData
// 	fields := []struct {
// 		key   string
// 		value string
// 	}{
// 		{"addr:city", place["addr:city"]},
// 		{"addr:street", place["addr:street"]},
// 		{"addr:housenumber", place["addr:housenumber"]},
// 		{"name", place["name"]},
// 		{"amenity", place["amenity"]},
// 		{"tourism", place["tourism"]},
// 		{"highway", place["highway"]},
// 		{"leisure", place["leisure"]},
// 		{"building", place["building"]},
// 		{"inscription", place["inscription"]},
// 		{"description", place["description"]},
// 	}

// 	for i, field := range fields {
// 		if i > 0 {
// 			jsonBuilder.WriteString(",")
// 		}
// 		// Экранируем значение для JSON
// 		escapedValue := strings.ReplaceAll(field.value, `"`, `\"`)
// 		jsonBuilder.WriteString(fmt.Sprintf(`"%s":"%s"`, field.key, escapedValue))
// 	}

// 	jsonBuilder.WriteString("}")
// 	jsonBody := jsonBuilder.String()

// 	req, err := http.NewRequestWithContext(ctxWithTimeout, "POST", os.Getenv("HOST_MISTRAL"), bytes.NewBufferString(jsonBody))
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
// 		Text string `json:"message"`
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
