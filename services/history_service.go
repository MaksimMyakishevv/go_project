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
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if len(places) == 0 {
		return "", fmt.Errorf("массив мест пуст")
	}

	// Формируем JSON вручную с помощью strings.Builder
	var jsonBuilder strings.Builder
	jsonBuilder.WriteString("[")

	// Поля для каждого места
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
		"place_name",
		"type",
	}

	for i, place := range places {
		if i > 0 {
			jsonBuilder.WriteString(",")
		}
		jsonBuilder.WriteString("{")
		for j, field := range fields {
			if value, exists := place[field]; exists && value != "" {
				if j > 0 {
					jsonBuilder.WriteString(",")
				}
				escapedValue := strings.ReplaceAll(value, `"`, `\"`)
				jsonBuilder.WriteString(fmt.Sprintf(`"%s":"%s"`, field, escapedValue))
			}
		}
		jsonBuilder.WriteString("}")
	}

	jsonBuilder.WriteString("]")
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

// ProcessPlaces обрабатывает массив мест последовательно
func (s *PlaceService) ProcessPlaces(userID uint, places []map[string]string) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"places":   []map[string]interface{}{},
		"response": nil,
		"audio":    nil,
		"status":   "pending",
	}
	ctx := context.Background()

	if len(places) == 0 {
		result["status"] = "success"
		return result, nil
	}

	// 1. Собираем информацию о местах
	placeList := []map[string]interface{}{}
	cacheFound := true

	for _, place := range places {
		placeName := place["place_name"]
		cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

		// Проверяем кеш
		if cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
			placeList = append(placeList, map[string]interface{}{
				"place_name": placeName,
				"details":    place,
			})
			result["response"] = cachedResponse
			result["audio"] = nil // Аудио не кешируется
		} else {
			cacheFound = false
			placeList = append(placeList, map[string]interface{}{
				"place_name": placeName,
				"details":    place,
			})
		}
	}

	result["places"] = placeList

	// Если все места найдены в кеше, возвращаем результат
	if cacheFound {
		result["status"] = "success"
		return result, nil
	}

	// 2. Отправляем весь массив в LLM
	text, err := s.SendBatchToLLM(places)
	if err != nil {
		result["status"] = "llm_error"
		result["response"] = err.Error()
		return result, err
	}

	result["response"] = text

	// 3. Генерируем аудио
	audioData, err := s.AudioGenerate(text)
	if err != nil {
		result["status"] = "tts_error"
		return result, err
	}

	result["audio"] = audioData
	result["status"] = "success"

	// 4. Сохраняем результаты в Redis и историю
	for _, place := range places {
		placeName := place["place_name"]
		cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

		// Сохраняем в Redis
		expiration := 24 * time.Hour
		if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
			fmt.Printf("Ошибка при сохранении в Redis: %v\n", err)
		}

		// Добавляем в историю
		_, err = s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
		if err != nil {
			fmt.Printf("Ошибка при добавлении в историю: %v\n", err)
			continue
		}
	}

	return result, nil
}

// ProcessPlacesGoroutines обрабатывает массив мест параллельно
func (s *PlaceService) ProcessPlacesGoroutines(userID uint, places []map[string]string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	ctx := context.Background()
	var mu sync.Mutex
	var wg sync.WaitGroup

	if len(places) == 0 {
		return results, nil
	}

	// Проверяем кеш для каждого места
	for _, place := range places {
		placeName := place["place_name"]
		cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

		if cachedResponse, err := database.RedisClient.Get(ctx, cacheKey).Result(); err == nil {
			mu.Lock()
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "success",
				"response":   cachedResponse,
				"audio":      nil,
			})
			mu.Unlock()
		}
	}

	// Если все места найдены в кеше, возвращаем результаты
	if len(results) == len(places) {
		return results, nil
	}

	// Отправляем весь массив в LLM в отдельной горутине
	var text string
	var llmErr error
	llmDone := make(chan struct{}) // Канал для сигнализации завершения LLM
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(llmDone) // Сигнализируем, что LLM завершился
		text, llmErr = s.SendBatchToLLM(places)
	}()

	// Генерируем аудио в отдельной горутине
	var audioData []byte
	var ttsErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Ждем завершения LLM через канал
		<-llmDone
		if llmErr != nil {
			return
		}
		audioData, ttsErr = s.AudioGenerate(text)
	}()

	// Ждем завершения LLM и TTS
	wg.Wait()

	// Обрабатываем ошибки LLM
	if llmErr != nil {
		mu.Lock()
		for _, place := range places {
			placeName := place["place_name"]
			if _, exists := findResultByPlaceName(results, placeName); exists {
				continue
			}
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "llm_error",
				"response":   llmErr.Error(),
				"audio":      nil,
			})
		}
		mu.Unlock()
		return results, llmErr
	}

	// Обрабатываем ошибки TTS
	if ttsErr != nil {
		mu.Lock()
		for _, place := range places {
			placeName := place["place_name"]
			if _, exists := findResultByPlaceName(results, placeName); exists {
				continue
			}
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "tts_error",
				"response":   text,
				"audio":      nil,
			})
		}
		mu.Unlock()
		return results, ttsErr
	}

	// Сохраняем результаты для каждого места в горутинах
	for _, place := range places {
		placeName := place["place_name"]
		cacheKey := fmt.Sprintf("llm:user:%d:place:%s", userID, placeName)

		// Пропускаем, если уже есть в кеше
		if _, exists := findResultByPlaceName(results, placeName); exists {
			continue
		}

		wg.Add(1)
		go func(placeName, cacheKey string) {
			defer wg.Done()

			// Сохраняем в Redis
			expiration := 24 * time.Hour
			if err := database.RedisClient.Set(ctx, cacheKey, text, expiration).Err(); err != nil {
				fmt.Printf("Ошибка при сохранении в Redis: %v\n", err)
			}

			// Добавляем в историю
			_, err := s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results = append(results, map[string]interface{}{
					"place_name": placeName,
					"status":     "failed_to_add",
					"response":   text,
					"audio":      audioData,
				})
				return
			}

			// Успешный результат
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "success",
				"response":   text,
				"audio":      audioData,
			})
		}(placeName, cacheKey)
	}

	wg.Wait()
	return results, nil
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
func (s *PlaceService) ProcessJSON(userID uint, osmObjects []dto.OSMObject) (map[string]interface{}, error) {
	places := make([]map[string]string, 0)
	placeList := make([]map[string]interface{}, 0)

	// Формируем список мест
	for _, obj := range osmObjects {
		// Пропускаем пустые места
		if isPlaceEmpty(obj.Tags) {
			continue
		}

		// Формируем выходное представление
		placeOutput := buildPlaceOutput(obj)
		placeList = append(placeList, placeOutput)

		// Формируем данные для ProcessPlaces
		placeData := buildPlaceData(obj)
		places = append(places, placeData)
	}

	// Инициализируем результат
	result := map[string]interface{}{
		"places":   placeList,
		"response": nil,
		"audio":    nil,
		"status":   "pending",
	}

	// Если нет мест, возвращаем пустой результат
	if len(places) == 0 {
		result["status"] = "success"
		return result, nil
	}

	// Обрабатываем места
	procResult, err := s.ProcessPlaces(userID, places)
	if err != nil {
		result["response"] = err.Error()
		result["status"] = procResult["status"]
		return result, err
	}

	// Обновляем результат
	result["response"] = procResult["response"]
	result["audio"] = procResult["audio"]
	result["status"] = procResult["status"]

	return result, nil
}

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
