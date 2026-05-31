package knowledge_pipeline

import "memoryflow/internal/ai/loader"

func NewLoader(memoryService loader.MemoryService) *loader.MemoryLoader {
	return loader.NewMemoryLoader(memoryService)
}
