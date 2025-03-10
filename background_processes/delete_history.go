package backgroundprocesses

import (
	"fmt"
	"new/models"
	"time"

	"gorm.io/gorm"
)

type Deletehistory struct {
	DB *gorm.DB
}

// CleanupOldPlaces удаляет записи старше 1 часа
func (s *Deletehistory) CleanupOldPlaces() {
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
