package retriever

import (
	"context"
	"errors"
	"memoryflow/internal/ai/component/vectorstore"
	"memoryflow/internal/domain/model"
	"strings"
	"time"
)

type RetrievedMemory struct {
	Memory model.MemoryItem `json:"memory"`
	Score  float32          `json:"score"`
}

type RetrieveOptions struct {
	TopK      int
	Type      string
	StartTime *time.Time
	EndTime   *time.Time
}

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type VectorStore interface {
	SearchMemoryVector(ctx context.Context, vector []float32, opt vectorstore.SearchOptions) ([]vectorstore.SearchResult, error)
}

type MemoryService interface {
	FindByIDs(ctx context.Context, ids []uint) ([]model.MemoryItem, error)
}

type MemoryRetriever struct {
	embeddingClient EmbeddingClient
	milvusStore     VectorStore
	memoryService   MemoryService
}

func NewMemoryRetriever(embeddingClient EmbeddingClient, milvusStore VectorStore, memoryService MemoryService) *MemoryRetriever {
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

	//2.time filter->unix timestamp
	var startUnix *int64
	if opt.StartTime != nil {
		v := opt.StartTime.Unix()
		startUnix = &v
	}

	var endUnix *int64
	if opt.EndTime != nil {
		v := opt.EndTime.Unix()
		endUnix = &v
	}

	//3. embedding -> Milvus topK search with metadata filters
	hits, err := r.milvusStore.SearchMemoryVector(ctx, queryVector, vectorstore.SearchOptions{
		TopK:      topK,
		Type:      opt.Type,
		StartUnix: startUnix,
		EndUnix:   endUnix,
	})
	if err != nil {
		return nil, err
	}

	if len(hits) == 0 {
		return []RetrievedMemory{}, nil
	}

	//4.collect memory ids
	ids := make([]uint, 0, len(hits))
	scoreMap := make(map[uint]float32, len(hits))

	for _, hit := range hits {
		id := uint(hit.MemoryID)
		ids = append(ids, id)
		scoreMap[id] = hit.Score
	}

	//5.memory_id -> SQLite full MemoryItem
	memories, err := r.memoryService.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	//6.merge memory+score
	results := make([]RetrievedMemory, 0, len(memories))
	for _, memory := range memories {
		results = append(results, RetrievedMemory{
			Memory: memory,
			Score:  scoreMap[memory.ID],
		})
	}

	return results, nil
}
