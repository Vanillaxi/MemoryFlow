package indexer

import (
	"context"

	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/domain/model"
)

type IndexDocument struct {
	MemoryID   int64
	Content    string
	MemoryType string
	OccurredAt int64
	Memory     model.MemoryItem
}

type MemoryIndexer struct {
	embeddingClient EmbeddingClient
	milvusStore     VectorStore
}

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type VectorStore interface {
	DeleteMemoryVector(ctx context.Context, memoryID int64) error
	InsertMemoryVector(ctx context.Context, item vectorstore.MemoryVector) error
}

func NewMemoryIndexer(embeddingClient EmbeddingClient, milvusStore VectorStore) *MemoryIndexer {
	return &MemoryIndexer{
		embeddingClient: embeddingClient,
		milvusStore:     milvusStore,
	}
}

func (i *MemoryIndexer) Index(ctx context.Context, doc IndexDocument) error {
	vec, err := i.embeddingClient.Embed(ctx, doc.Content)
	if err != nil {
		return err
	}

	_ = i.milvusStore.DeleteMemoryVector(ctx, doc.MemoryID)

	return i.milvusStore.InsertMemoryVector(ctx, vectorstore.MemoryVector{
		MemoryID:   doc.MemoryID,
		Content:    truncateForMilvus(doc.Content, 4000),
		MemoryType: doc.MemoryType,
		OccurredAt: doc.OccurredAt,
		Vector:     vec,
	})
}

func truncateForMilvus(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}
