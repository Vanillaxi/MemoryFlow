package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Model     ModelConfig     `mapstructure:"model"`
	Embedding EmbeddingConfig `mapstructure:"embedding"`
	Milvus    MilvusConfig    `mapstructure:"milvus"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"host"`
	DSN    string `mapstructure:"dsn"`
}

type StorageConfig struct {
	UploadDir string `mapstructure:"upload_dir"`
}

type ModelConfig struct {
	BaseURL   string `mapstructure:"base_url"`
	APIKey    string `mapstructure:"api_key"`
	ModelName string `mapstructure:"model_name"`
}

type EmbeddingConfig struct {
	BaseURL   string `mapstructure:"base_url"`
	APIKey    string `mapstructure:"api_key"`
	ModelName string `mapstructure:"model_name"`
	Dim       int    `mapstructure:"dim"`
}

type MilvusConfig struct {
	Address    string `mapstructure:"address"`
	Collection string `mapstructure:"collection"`
}

func LoadConfig(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "../memoryflow-data/data/memoryflow.db")
	v.SetDefault("stroage.upload_dir", "../memoryflow-data/uploads")

	for key, env := range map[string]string{
		"server.host":    "SERVER_HOST",
		"server.port":    "SERVER_PORT",
		"milvus.address": "MILVUS_ADDRESS",
	} {
		if err := v.BindEnv(key, env); err != nil {
			return nil, fmt.Errorf("bind config env %s failed: %w", strings.ToUpper(env), err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config failed: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}
	return &cfg, nil
}
