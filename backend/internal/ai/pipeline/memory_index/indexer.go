package memory_index

import (
	"context"

	"memoryflow/internal/ai/component/vectorstore"
)

type Indexer struct {
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

func NewIndexer(
	embeddingClient EmbeddingClient,
	milvusStore VectorStore,
) *Indexer {
	return &Indexer{
		embeddingClient: embeddingClient,
		milvusStore:     milvusStore,
	}
}

func (i *Indexer) Index(ctx context.Context, doc IndexDocument) error {
	vec, err := i.embeddingClient.Embed(ctx, doc.Content)
	if err != nil {
		return err
	}

	// 先删旧向量，再插入新向量，避免重复
	_ = i.milvusStore.DeleteMemoryVector(ctx, doc.MemoryID)

	return i.milvusStore.InsertMemoryVector(ctx, vectorstore.MemoryVector{
		MemoryID:   doc.MemoryID,
		Content:    truncateForMilvus(doc.Content, 4000),
		MemoryType: doc.MemoryType,
		OccurredAt: doc.OccurredAt,
		Vector:     vec,
	})
}
