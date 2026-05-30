package models

import "time"

type Config struct {
	BaseURL     string
	APIKey      string
	ModelName   string
	HTTPTimeout time.Duration
}
