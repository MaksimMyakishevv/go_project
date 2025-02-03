package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io" // Заменяем "io/ioutil" на "io"
	"net/http"
	"time"

	"new/dto"
	"new/models"

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

// GetUserHistory возвращает историю запросов пользователя
func (s *PlaceService) GetUserHistory(userID uint) ([]models.Place, error) {
	var places []models.Place
	if err := s.DB.Where("user_id = ?", userID).Find(&places).Error; err != nil {
		return nil, err
	}
	return places, nil
}

// ProcessPlace отправляет место на обработку нейросети с таймаутом 5 секунд
func (s *PlaceService) ProcessPlace(placeName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := &http.Client{}
	reqBody := map[string]string{"place_name": placeName}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://neural-network-service.com/process", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body) // Заменяем ioutil.ReadAll на io.ReadAll
	if err != nil {
		return "", err
	}

	var neuralResponse struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(body, &neuralResponse); err != nil {
		return "", err
	}

	return neuralResponse.Result, nil
}

// ProcessPlaces обрабатывает массив мест и отправляет их на обработку нейросетью
func (s *PlaceService) ProcessPlaces(userID uint, places []string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	for _, placeName := range places {
		// Добавляем место в историю пользователя
		_, err := s.AddPlace(userID, dto.AddPlaceDTO{PlaceName: placeName})
		if err != nil {
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "failed_to_add",
				"response":   nil,
			})
			continue
		}

		// Отправляем запрос на нейросеть с таймаутом 5 секунд
		result, err := s.ProcessPlace(placeName)
		if err != nil {
			results = append(results, map[string]interface{}{
				"place_name": placeName,
				"status":     "timeout",
				"response":   nil,
			})
			continue
		}

		// Сохраняем результат
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
