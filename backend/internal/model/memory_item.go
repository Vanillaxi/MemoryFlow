package model

import "time"

type MemoryItem struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Type             string     `gorm:"not null;index" json:"type"` // text/image/mixed
	ContentText      string     `json:"content_text"`
	ImageURL         string     `json:"image_URL"`
	OccurredAt       time.Time  `gorm:"index" json:"occurred_at"`
	Location         string     `json:"location"`
	Summary          string     `json:"summary"`
	Mood             string     `json:"mood"`
	Tags             string     `json:"tags"` // JSON string
	ImportanceSource float64    `json:"importance_score"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at"`
}
