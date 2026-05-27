package model

import "time"

type Task struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	TaskType     string `gorm:"not null;index" json:"task_type"` //text_analyze,image_analyze,embedding,report_generate
	Status       string `gorm:"not null;index" json:"status"`    //pending,runnung,success,fail
	TargetID     uint   `gorm:"index" json:"target_id"`
	ErrorMessage string `json:"error_message"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
