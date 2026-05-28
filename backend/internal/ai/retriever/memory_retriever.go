package retriever

import (
	"context"
	"errors"
	"memoryflow/internal/ai/embedding"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/model"
	"memoryflow/internal/service"
	"strings"
)

type RetrievedMemory struct {
	Memory model.MemoryItem `json:"memory"`
	Score  float32          `json:"score"`
}

type RetrieveOptions struct {
	TopK int
}

type MemoryRetriever struct {
	embeddingClient *embedding.Client
	milvusStore     *vectorstore.MilvusStore
	memoryService   *service.MemoryService
}

func NewMemoryRettriever(embeddingClient *embedding.Client, milvusStore *vectorstore.MilvusStore, memoryService *service.MemoryService) *MemoryRetriever {
	return &MemoryRetriever{
		embeddingClient: embeddingClient,
		milvusStore:     milvusStore,
		memoryService:   memoryService,
	}
}

func (r *MemoryRetriever) Retrieve(ctx context.Context, query string, opt RetrieveOptions) ([]RetrievedMemory, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}

	topK := opt.TopK
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	//1.query->embedding
	queryVector, err := r.embeddingClient.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	//2.embedding->Milvus topK search
	hits, err := r.milvusStore.SearchMemoryVector(ctx, queryVector, topK)
	if err != nil {
		return nil, err
	}

	if len(hits) == 0 {
		return []RetrievedMemory{}, nil
	}

	//3.collect memory ids
	ids := make([]uint, 0, len(hits))
	scoreMap := make(map[uint]float32, len(hits))

	for _, hit := range hits {
		id := uint(hit.MemoryID)
		ids = append(ids, id)
		scoreMap[id] = hit.Score
	}

	//4.memory_id -> SQLite full MemoryItem
	memories, err := r.memoryService.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	//5.merge memory+score
	results := make([]RetrievedMemory, 0, len(memories))
	for _, memory := range memories {
		results = append(results, RetrievedMemory{
			Memory: memory,
			Score:  scoreMap[memory.ID],
		})
	}

	return results, nil
}
