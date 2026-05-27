package repository

import (
	"fmt"
	"memoryflow/internal/model"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitSQLite(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("sqlite dsn is empty")
	}

	abs, err := filepath.Abs(dsn)
	if err != nil {
		return nil, err
	}

	//如果 memoryflow-data/data 不存在，也会自动创建。
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(abs), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&model.MemoryItem{},
		&model.Task{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
