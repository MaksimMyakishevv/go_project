package models

type Preference struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	UserID           uint           `json:"user_id" gorm:"not null;index"` // Внешний ключ для связи с User
	ListPreferenceID uint           `json:"list_preference_id" gorm:"not null;index"`
	ListPreference   ListPreference `json:"list_preference" gorm:"foreignKey:ListPreferenceID;constraint:onUpdate:CASCADE,onDelete:RESTRICT"`
	User             User           `json:"-" gorm:"foreignKey:UserID;constraint:onUpdate:CASCADE,onDelete:CASCADE"`
}
