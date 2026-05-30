package retriever

import (
	"context"
	"errors"
	"testing"

	"memoryflow/internal/ai/vectorstore"
	"memoryflow/internal/domain/model"
)

type fakeEmbeddingClient struct {
	err error
}

func (f *fakeEmbeddingClient) Embed(_ context.Context, _ string) ([]float32, error) {
	return []float32{1, 2, 3}, f.err
}

type fakeVectorStore struct {
	opt  vectorstore.SearchOptions
	hits []vectorstore.SearchResult
	err  error
}

func (f *fakeVectorStore) SearchMemoryVector(_ context.Context, _ []float32, opt vectorstore.SearchOptions) ([]vectorstore.SearchResult, error) {
	f.opt = opt
	return f.hits, f.err
}

type fakeMemoryService struct {
	ids      []uint
	memories []model.MemoryItem
	err      error
}

func (f *fakeMemoryService) FindByIDs(_ context.Context, ids []uint) ([]model.MemoryItem, error) {
	f.ids = ids
	return f.memories, f.err
}

func TestMemoryRetrieverRejectsEmptyQuery(t *testing.T) {
	r := NewMemoryRetriever(&fakeEmbeddingClient{}, &fakeVectorStore{}, &fakeMemoryService{})
	if _, err := r.Retrieve(context.Background(), "  ", RetrieveOptions{}); err == nil {
		t.Fatal("expected empty query error")
	}
}

func TestMemoryRetrieverUsesDefaultTopKAndMergesScore(t *testing.T) {
	store := &fakeVectorStore{hits: []vectorstore.SearchResult{{MemoryID: 7, Score: 0.91}}}
	service := &fakeMemoryService{memories: []model.MemoryItem{{ID: 7, Summary: "found"}}}
	r := NewMemoryRetriever(&fakeEmbeddingClient{}, store, service)

	got, err := r.Retrieve(context.Background(), "query", RetrieveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if store.opt.TopK != 5 {
		t.Fatalf("TopK = %d, want 5", store.opt.TopK)
	}
	if len(got) != 1 || got[0].Memory.ID != 7 || got[0].Score != 0.91 {
		t.Fatalf("unexpected results: %#v", got)
	}
	if len(service.ids) != 1 || service.ids[0] != 7 {
		t.Fatalf("unexpected IDs: %#v", service.ids)
	}
}

func TestMemoryRetrieverEmbeddingError(t *testing.T) {
	r := NewMemoryRetriever(&fakeEmbeddingClient{err: errors.New("embed failed")}, &fakeVectorStore{}, &fakeMemoryService{})
	if _, err := r.Retrieve(context.Background(), "query", RetrieveOptions{}); err == nil {
		t.Fatal("expected embedding error")
	}
}

func TestMemoryRetrieverVectorStoreError(t *testing.T) {
	r := NewMemoryRetriever(&fakeEmbeddingClient{}, &fakeVectorStore{err: errors.New("search failed")}, &fakeMemoryService{})
	if _, err := r.Retrieve(context.Background(), "query", RetrieveOptions{}); err == nil {
		t.Fatal("expected search error")
	}
}

func TestMemoryRetrieverMissingMemoryIDDoesNotPanic(t *testing.T) {
	r := NewMemoryRetriever(
		&fakeEmbeddingClient{},
		&fakeVectorStore{hits: []vectorstore.SearchResult{{MemoryID: 99, Score: 0.5}}},
		&fakeMemoryService{memories: []model.MemoryItem{}},
	)
	got, err := r.Retrieve(context.Background(), "query", RetrieveOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("unexpected results: %#v", got)
	}
}
