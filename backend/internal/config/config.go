package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StroageConfig  `mapstructure:"stroage"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"host"`
	DSN    string `mapstructure:"dsn"`
}

type StroageConfig struct {
	UploadDir string `mapstructure:"upload_dir"`
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("server.port", 8080)
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "../memoryflow-data/data/memoryflow.db")
	v.SetDefault("stroage.upload_dir", "../memoryflow-data/uploads")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	return &cfg, nil
}
