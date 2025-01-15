package models

type Preference struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	UserID   uint   `json:"user_id" gorm:"not null;index"` // Внешний ключ для связи с User
	Place    string `json:"place" gorm:"not null"`         // Название места
	Priority int    `json:"priority" gorm:"default:0"`     // Приоритет
	User     User   `json:"-" gorm:"foreignKey:UserID;constraint:onUpdate:CASCADE,onDelete:CASCADE"`
}
