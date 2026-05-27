package main

import (
	"fmt"
	"log"
	"memoryflow/internal/api"
	"memoryflow/internal/config"
	"memoryflow/internal/repository"
	"memoryflow/internal/service"
	"memoryflow/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		log.Fatal("load config failed", err)
	}

	db, err := repository.InitSQLite(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("init sqlite failed: %v", err)
	}

	memoryRepo := repository.NewSQLiteMemoryRepository(db)
	memoryService := service.NewMemoryService(memoryRepo)

	localStorage := storage.NewLocalStorage(cfg.Storage.UploadDir)
	memoryHandler := api.NewMemoryHandler(memoryService, localStorage)

	r := gin.Default()
	api.RegisterRouters(r, memoryHandler)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}

}
