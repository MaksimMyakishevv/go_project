package models

import "time"

// Place представляет сущность места
type Place struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"not null;index"`               // Внешний ключ для связи с User
	PlaceName string    `json:"place_name" gorm:"not null"`                  // Название места
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"` // Время создания записи
	User      User      `json:"-" gorm:"foreignKey:UserID;constraint:onUpdate:CASCADE,onDelete:CASCADE"`
}
