package chat_pipeline

import "memoryflow/internal/ai/retriever"

func NewRetriever(
	embeddingClient retriever.EmbeddingClient,
	vectorStore retriever.VectorStore,
	memoryService retriever.MemoryService,
) *retriever.MemoryRetriever {
	return retriever.NewMemoryRetriever(embeddingClient, vectorStore, memoryService)
}
