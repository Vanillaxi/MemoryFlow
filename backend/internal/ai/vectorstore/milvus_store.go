package vectorstore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

const (
	FieldMemoryID   = "memory_id"
	FieldContent    = "content"
	FieldMemoryType = "memory_type"
	FieldOccurredAt = "occurred_at"
	FieldVector     = "vector"
)

type MilvusStore struct {
	client     *milvusclient.Client
	address    string
	collection string
	dim        int
}

type MemoryVector struct {
	MemoryID   int64
	Content    string
	MemoryType string
	OccurredAt int64
	Vector     []float32
}

type SearchResult struct {
	MemoryID int64
	Score    float32
}

func NewMilvusStore(ctx context.Context, address, collection string, dim int) (*MilvusStore, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("milvus address is required")
	}
	if strings.TrimSpace(collection) == "" {
		return nil, errors.New("milvus collection is required")
	}
	if dim <= 0 {
		return nil, errors.New("milvus vector dim must be positive")
	}

	cli, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	return &MilvusStore{
		client:     cli,
		address:    address,
		collection: collection,
		dim:        dim,
	}, nil

}

func (s *MilvusStore) EnsureCollection(ctx context.Context) error {
	exists, err := s.client.HasCollection(
		ctx,
		milvusclient.NewHasCollectionOption(s.collection),
	)
	if err != nil {
		return fmt.Errorf("check milvus collection failed: %w", err)
	}

	if exists {
		log.Printf("[milvus] collection %s already exists", s.collection)

		loadTask, err := s.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(s.collection))
		if err != nil {
			return fmt.Errorf("load existing collection failed: %w", err)
		}
		if err := loadTask.Await(ctx); err != nil {
			return fmt.Errorf("await load existing collection failed: %w", err)
		}

		log.Printf("[milvus] collection %s loaded\n", s.collection)
		return nil
	}

	log.Printf("[milvus] collection %s not found,creating...\n", s.collection)

	schema := entity.NewSchema().
		WithDynamicFieldEnabled(false).
		WithField(entity.NewField().
			WithName(FieldMemoryID).
			WithDataType(entity.FieldTypeInt64).
			WithIsPrimaryKey(true).
			WithIsAutoID(false),
		).
		WithField(entity.NewField().
			WithName(FieldContent).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(4096),
		).
		WithField(entity.NewField().
			WithName(FieldMemoryType).
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(32),
		).
		WithField(entity.NewField().
			WithName(FieldOccurredAt).
			WithDataType(entity.FieldTypeInt64),
		).
		WithField(entity.NewField().
			WithName(FieldVector).
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(s.dim)),
		)

	indexOptions := []milvusclient.CreateIndexOption{
		milvusclient.NewCreateIndexOption(
			s.collection,
			FieldVector,
			index.NewAutoIndex(entity.COSINE),
		).WithIndexName("idx_vector"),

		milvusclient.NewCreateIndexOption(
			s.collection,
			FieldMemoryID,
			index.NewSortedIndex(),
		).WithIndexName("idx_memory_id"),
	}

	err = s.client.CreateCollection(
		ctx,
		milvusclient.NewCreateCollectionOption(s.collection, schema).
			WithIndexOptions(indexOptions...),
	)
	if err != nil {
		return fmt.Errorf("create milvus collection failed: %w", err)
	}

	log.Printf("[milvus] collection %s created\n", s.collection)

	loadTask, err := s.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(s.collection))
	if err != nil {
		return fmt.Errorf("await load collection failed: %w", err)
	}
	if err := loadTask.Await(ctx); err != nil {
		return fmt.Errorf("await load collection failed: %w", err)
	}

	log.Printf("[milvus] collection %s loaded\n", s.collection)

	return nil
}

func (s *MilvusStore) Close(ctx context.Context) error {
	if s.client != nil {
		return s.client.Close(ctx)
	}
	return nil
}

func (s *MilvusStore) InsertMemoryVector(ctx context.Context, item MemoryVector) error {
	if item.MemoryID <= 0 {
		return errors.New("memory_id must be positive")
	}
	if strings.TrimSpace(item.MemoryType) == "" {
		return errors.New("memory_type is required")
	}
	if len(item.Vector) != s.dim {
		return fmt.Errorf("vector dim mismatch: got=%d,want=%d", len(item.Vector), s.dim)
	}

	_, err := s.client.Insert(
		ctx,
		milvusclient.NewColumnBasedInsertOption(s.collection).
			WithInt64Column(FieldMemoryID, []int64{item.MemoryID}).
			WithVarcharColumn(FieldContent, []string{item.Content}).
			WithVarcharColumn(FieldMemoryType, []string{item.MemoryType}).
			WithInt64Column(FieldOccurredAt, []int64{item.OccurredAt}).
			WithFloatVectorColumn(FieldVector, s.dim, [][]float32{item.Vector}),
	)
	if err != nil {
		return fmt.Errorf("insert memory vector failed: %w", err)
	}

	log.Printf("[milvus] insert memory vector successfully,memory_id=%d \n", item.MemoryID)
	return nil
}

func (s *MilvusStore) SearchMemoryVector(ctx context.Context, vector []float32, topK int) ([]SearchResult, error) {
	if len(vector) != s.dim {
		return nil, fmt.Errorf("query vector dim mismatch: got=%d, want=%d", len(vector), s.dim)
	}
	if topK <= 0 {
		topK = 5
	}

	resultSets, err := s.client.Search(
		ctx,
		milvusclient.NewSearchOption(
			s.collection,
			topK,
			[]entity.Vector{entity.FloatVector(vector)},
		).
			WithANNSField(FieldVector).
			WithOutputFields(FieldMemoryID).
			WithConsistencyLevel(entity.ClBounded),
	)
	if err != nil {
		return nil, fmt.Errorf("search memory vector failed: %w", err)
	}

	results := make([]SearchResult, 0)

	for _, resultSet := range resultSets {
		for i := 0; i < resultSet.ResultCount; i++ {
			id, err := resultSet.GetColumn(FieldMemoryID).GetAsInt64(i)
			if err != nil {
				return nil, err
			}

			score := resultSet.Scores[i]
			results = append(results, SearchResult{
				MemoryID: id,
				Score:    score,
			})
		}
	}
	return results, nil
}
