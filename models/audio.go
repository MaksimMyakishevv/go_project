package models

import "time"

type Audio struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Path string    `json:"path" gorm:"not null"`                  // Путь
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"` // Время создания записи
}