package services

import (
	// "errors"
	"new/dto"
	"new/models"

	"gorm.io/gorm"
)

type AudioService struct {
	DB *gorm.DB
}


func (s *AudioService) SaveAudio(input dto.AudioDTO) (*models.Audio, error) {

	file_path := &models.Audio{
		Path:    input.Path,
	}

	// Сохраняем в базе данных
	if err := s.DB.Create(file_path).Error; err != nil {
		return nil, err
	}

	return file_path, nil
}

func (s *AudioService) GetAllAudio() ([]models.Audio, error) {
	var audio []models.Audio

	if err := s.DB.Find(&audio).Error; err != nil {
		return nil, err
	}

	return audio, nil
}
