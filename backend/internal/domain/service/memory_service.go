package service

import (
	"context"
	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/repository"
	"time"
)

type CreateTextMemoryRequest struct {
	ContentText string    `json:"content_text"`
	OccurredAt  time.Time `json:"occurred_at"`
	Location    string    `json:"location"`
}

type CreateImageMemoryRequest struct {
	ContentText string
	ImageURL    string
	OccurredAt  time.Time
	Location    string
}

type MemoryService struct {
	repo repository.MemoryRepository
}

type TimelineGroup struct {
	Date  string             `json:"date"`
	Items []model.MemoryItem `json:"items"`
}

func NewMemoryService(repo repository.MemoryRepository) *MemoryService {
	return &MemoryService{repo: repo}
}

func (s *MemoryService) CreateTextMemory(ctx context.Context, req *CreateTextMemoryRequest) (*model.MemoryItem, error) {
	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	item := &model.MemoryItem{
		Type:        "text",
		ContentText: req.ContentText,
		OccurredAt:  occurredAt,
		Location:    req.Location,
		Tags:        "[]",
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *MemoryService) CreateImageMemory(ctx context.Context, req *CreateImageMemoryRequest) (*model.MemoryItem, error) {
	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	memoryType := "image"
	if req.ContentText != "" {
		memoryType = "mixed"
	}

	item := &model.MemoryItem{
		Type:        memoryType,
		ContentText: req.ContentText,
		ImageURL:    req.ImageURL,
		OccurredAt:  occurredAt,
		Location:    req.Location,
		Tags:        "[]",
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *MemoryService) ListRecent(ctx context.Context, limit int) ([]model.MemoryItem, error) {
	return s.repo.FindRecent(ctx, limit)
}

func (s *MemoryService) ListByTimeRange(ctx context.Context, from, to time.Time, limit int) ([]*model.MemoryItem, error) {
	return s.repo.ListByTimeRange(ctx, from, to, limit)
}

// 时间线默认从新到旧
func (s *MemoryService) GetTimeline(ctx context.Context, start, end time.Time) ([]TimelineGroup, error) {
	items, err := s.repo.FindByTimeRange(ctx, start, end, 500)
	if err != nil {
		return nil, err
	}

	groupMap := make(map[string][]model.MemoryItem)
	dateOrder := make([]string, 0)

	for _, item := range items {
		date := item.OccurredAt.Format("2006-01-02")

		if _, ok := groupMap[date]; !ok {
			dateOrder = append(dateOrder, date)
		}

		groupMap[date] = append(groupMap[date], item)
	}

	groups := make([]TimelineGroup, 0, len(groupMap))
	for _, date := range dateOrder {
		groups = append(groups, TimelineGroup{
			Date:  date,
			Items: groupMap[date],
		})
	}

	return groups, nil
}

func (s *MemoryService) GetByID(ctx context.Context, id uint) (*model.MemoryItem, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *MemoryService) UpdateAnalysis(ctx context.Context, id uint, summary string, tags string, mood string, importanceScore float64) error {
	return s.repo.UpdateAnalysis(ctx, id, summary, tags, mood, importanceScore)
}

func (s *MemoryService) FindByIDs(ctx context.Context, ids []uint) ([]model.MemoryItem, error) {
	return s.repo.FindByIDs(ctx, ids)
}

func (s *MemoryService) ListForIndex(ctx context.Context, limit int, offset int) ([]model.MemoryItem, error) {
	return s.repo.ListForIndex(ctx, limit, offset)
}
