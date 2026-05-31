package knowledge_pipeline

import (
	"context"
	"errors"
	"testing"

	"memoryflow/internal/ai/indexer"
	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/domain/model"
)

type fakeIndexMemoryService struct {
	items []model.MemoryItem
}

func (f *fakeIndexMemoryService) LoadForIndex(_ context.Context, limit int, offset int) ([]model.MemoryItem, error) {
	if offset >= len(f.items) {
		return []model.MemoryItem{}, nil
	}
	end := offset + limit
	if end > len(f.items) {
		end = len(f.items)
	}
	return f.items[offset:end], nil
}

type fakeDocumentIndexer struct {
	failID int64
	docs   []indexer.IndexDocument
}

func (f *fakeDocumentIndexer) Index(_ context.Context, doc indexer.IndexDocument) error {
	f.docs = append(f.docs, doc)
	if doc.MemoryID == f.failID {
		return errors.New("index failed")
	}
	return nil
}

type fakeIndexEmbedding struct {
	err error
}

func (f *fakeIndexEmbedding) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.1, 0.2}, f.err
}

type fakeIndexVectorStore struct {
	deleted  []int64
	inserted []vectorstore.MemoryVector
}

func (f *fakeIndexVectorStore) DeleteMemoryVector(_ context.Context, memoryID int64) error {
	f.deleted = append(f.deleted, memoryID)
	return nil
}

func (f *fakeIndexVectorStore) InsertMemoryVector(_ context.Context, item vectorstore.MemoryVector) error {
	f.inserted = append(f.inserted, item)
	return nil
}

func TestReindexAllCountsSuccessAndFailure(t *testing.T) {
	service := &fakeIndexMemoryService{items: []model.MemoryItem{{ID: 1}, {ID: 2}, {ID: 3}}}
	indexer := &fakeDocumentIndexer{failID: 2}
	pipeline := NewPipeline(service, indexer)

	got, err := pipeline.ReindexAll(context.Background(), ReindexInput{BatchSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 3 || got.Succeeded != 2 || got.Failed != 1 {
		t.Fatalf("unexpected output: %#v", got)
	}
	if len(indexer.docs) != 3 {
		t.Fatalf("indexed docs = %d, want 3", len(indexer.docs))
	}
}

func TestIndexerUsesEmbeddingAndVectorStore(t *testing.T) {
	store := &fakeIndexVectorStore{}
	idx := indexer.NewMemoryIndexer(&fakeIndexEmbedding{}, store)

	if err := idx.Index(context.Background(), indexer.IndexDocument{MemoryID: 9, Content: "content", MemoryType: "text"}); err != nil {
		t.Fatal(err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != 9 || len(store.inserted) != 1 {
		t.Fatalf("unexpected store calls: deleted=%#v inserted=%#v", store.deleted, store.inserted)
	}
	if len(store.inserted[0].Vector) != 2 {
		t.Fatalf("unexpected vector: %#v", store.inserted[0].Vector)
	}
}

func TestIndexerEmbeddingErrorStopsStoreWrites(t *testing.T) {
	store := &fakeIndexVectorStore{}
	idx := indexer.NewMemoryIndexer(&fakeIndexEmbedding{err: errors.New("embed failed")}, store)
	if err := idx.Index(context.Background(), indexer.IndexDocument{MemoryID: 9, Content: "content"}); err == nil {
		t.Fatal("expected embedding error")
	}
	if len(store.deleted) != 0 || len(store.inserted) != 0 {
		t.Fatalf("unexpected store calls: deleted=%#v inserted=%#v", store.deleted, store.inserted)
	}
}
