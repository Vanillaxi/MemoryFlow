package vectorstore

import (
	"context"
	"errors"
)

// Milvus 不可用时的占位向量库
var ErrVectorStoreDisabled = errors.New("vector store is disabled: milvus is not available")

type DisabledStore struct{}

func NewDisabledStore() *DisabledStore {
	return &DisabledStore{}
}

func (s *DisabledStore) SearchMemoryVector(context.Context, []float32, SearchOptions) ([]SearchResult, error) {
	return nil, ErrVectorStoreDisabled
}

func (s *DisabledStore) DeleteMemoryVector(context.Context, int64) error {
	return ErrVectorStoreDisabled
}

func (s *DisabledStore) InsertMemoryVector(context.Context, MemoryVector) error {
	return ErrVectorStoreDisabled
}
