package services

import (
	"new/dto"
	"new/models"

	"gorm.io/gorm"
)

type PreferenceService struct {
	DB *gorm.DB
}

func NewPreferenceService(db *gorm.DB) *PreferenceService {
	return &PreferenceService{DB: db}
}

// CreatePreference добавляет новое предпочтение для пользователя
func (s *PreferenceService) CreatePreference(userID uint, input dto.CreatePreferenceDTO) (*models.Preference, error) {
	// Создаем новое предпочтение
	preference := &models.Preference{
		UserID:         userID,
		ListPreference: models.ListPreference{},
	}

	// Сохраняем предпочтение в базе данных
	if err := s.DB.Create(preference).Error; err != nil {
		return nil, err
	}

	return preference, nil
}

// GetPreferencesByUserID возвращает список предпочтений пользователя
func (s *PreferenceService) GetPreferencesByUserID(userID uint) ([]models.Preference, error) {
	var preferences []models.Preference

	if err := s.DB.Where("user_id = ?", userID).Find(&preferences).Error; err != nil {
		return nil, err
	}

	return preferences, nil
}

// DeletePreference удаляет предпочтение по ID и UserID
func (s *PreferenceService) DeletePreference(userID, preferenceID uint) error {
	if err := s.DB.Where("id = ? AND user_id = ?", preferenceID, userID).Delete(&models.Preference{}).Error; err != nil {
		return err
	}

	return nil
}
