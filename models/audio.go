package models

import (
	"io"
	"time"
)

type Audio struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Path string    `json:"path" gorm:"not null"`                  // Путь
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"` // Время создания записи
}

type UploadedFile struct {
	File io.Reader
	Filename string
	Size int64
}

type TTSRequest struct {
	Text string `json:"text"`
}