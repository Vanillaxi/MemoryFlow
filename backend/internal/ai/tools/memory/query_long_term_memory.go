package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"memoryflow/internal/ai/retriever"
	"memoryflow/internal/domain/model"
)

func QueryLongTermMemory(
	ctx context.Context,
	memoryRetriever MemoryRetriever,
	memoryService MemoryService,
	input QueryLongTermMemoryInput,
) (*QueryLongTermMemoryOutput, error) {
	mode := resolveMode(input)
	from, to, err := parseDateRange(input.From, input.To)
	if err != nil {
		return nil, err
	}
	limit := normalizeLimit(input.Limit)

	switch mode {
	case ModeSemantic:
		query := strings.TrimSpace(input.Query)
		if query == "" {
			return nil, fmt.Errorf("query is required for semantic mode")
		}
		evidence, err := memoryRetriever.Retrieve(ctx, query, retriever.RetrieveOptions{
			TopK:      limit,
			StartTime: from,
			EndTime:   to,
		})
		if err != nil {
			return nil, err
		}
		return &QueryLongTermMemoryOutput{
			Mode:     mode,
			Evidence: toEvidence(evidence),
		}, nil
	case ModeTimeline, ModeAggregate:
		from, to, err = defaultDateRange(from, to)
		if err != nil {
			return nil, err
		}
		memories, err := memoryService.ListByTimeRange(ctx, *from, *to, limit)
		if err != nil {
			return nil, err
		}
		output := &QueryLongTermMemoryOutput{
			Mode:  mode,
			Items: toItems(memories),
		}
		if mode == ModeAggregate {
			aggregation := AggregateMemories(memories)
			output.Aggregation = &LongTermMemoryAggregation{
				Count:      aggregation.Count,
				Tags:       aggregation.Tags,
				Moods:      aggregation.Moods,
				Highlights: truncateHighlights(aggregation.Highlights),
			}
		}
		return output, nil
	default:
		return nil, fmt.Errorf("invalid mode, expected semantic/timeline/aggregate")
	}
}

func resolveMode(input QueryLongTermMemoryInput) string {
	if mode := strings.TrimSpace(input.Mode); mode != "" {
		return mode
	}
	if strings.TrimSpace(input.Query) != "" {
		return ModeSemantic
	}
	return ModeTimeline
}

func parseDateRange(fromText, toText string) (*time.Time, *time.Time, error) {
	from, err := parseDate("from", fromText, false)
	if err != nil {
		return nil, nil, err
	}
	to, err := parseDate("to", toText, true)
	if err != nil {
		return nil, nil, err
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, nil, fmt.Errorf("from must not be after to")
	}
	return from, to, nil
}

func parseDate(name, value string, includeWholeDay bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s format, expected YYYY-MM-DD", name)
	}
	if includeWholeDay {
		parsed = parsed.Add(24*time.Hour - time.Second)
	}
	return &parsed, nil
}

func defaultDateRange(from, to *time.Time) (*time.Time, *time.Time, error) {
	if from == nil && to == nil {
		now := time.Now()
		start := now.AddDate(0, 0, -7)
		return &start, &now, nil
	}
	if from == nil || to == nil {
		return nil, nil, fmt.Errorf("from and to must be provided together for timeline and aggregate modes")
	}
	return from, to, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

func toEvidence(evidence []retriever.RetrievedMemory) []LongTermMemoryEvidence {
	items := make([]LongTermMemoryEvidence, 0, len(evidence))
	for _, item := range evidence {
		items = append(items, LongTermMemoryEvidence{
			LongTermMemoryItem: toItem(&item.Memory),
			Score:              item.Score,
		})
	}
	return items
}

func toItems(memories []*model.MemoryItem) []LongTermMemoryItem {
	items := make([]LongTermMemoryItem, 0, len(memories))
	for _, memory := range memories {
		if memory != nil {
			items = append(items, toItem(memory))
		}
	}
	return items
}

func toItem(memory *model.MemoryItem) LongTermMemoryItem {
	return LongTermMemoryItem{
		MemoryID:        memory.ID,
		Type:            memory.Type,
		OccurredAt:      memory.OccurredAt,
		Location:        memory.Location,
		Summary:         truncateSummary(memory.Summary),
		Mood:            memory.Mood,
		Tags:            memory.Tags,
		ImportanceScore: memory.ImportanceScore,
	}
}

func truncateSummary(summary string) string {
	runes := []rune(strings.TrimSpace(summary))
	if len(runes) <= MaxSummaryLength {
		return string(runes)
	}
	return string(runes[:MaxSummaryLength]) + "..."
}

func truncateHighlights(highlights []string) []string {
	items := make([]string, 0, len(highlights))
	for _, highlight := range highlights {
		items = append(items, truncateSummary(highlight))
	}
	return items
}
