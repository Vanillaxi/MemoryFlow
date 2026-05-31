package knowledge_pipeline

import "memoryflow/internal/ai/indexer"

func NewIndexer(
	embeddingClient indexer.EmbeddingClient,
	vectorStore indexer.VectorStore,
) *indexer.MemoryIndexer {
	return indexer.NewMemoryIndexer(embeddingClient, vectorStore)
}
