package memory_summary

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"memoryflow/internal/domain/model"
)

const (
	maxMemoryListContent = 300
	maxHighlights        = 5
	maxAggregationValues = 10
)

func AggregateMemories(memories []*model.MemoryItem) SummaryAggregation {
	items := make([]*model.MemoryItem, 0, len(memories))
	for _, item := range memories {
		if item != nil {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].OccurredAt.Before(items[j].OccurredAt)
	})

	tagCounts := make(map[string]int)
	moodCounts := make(map[string]int)
	for _, item := range items {
		for _, tag := range parseTags(item.Tags) {
			tagCounts[tag]++
		}
		if mood := strings.TrimSpace(item.Mood); mood != "" {
			moodCounts[mood]++
		}
	}

	highlightItems := append([]*model.MemoryItem(nil), items...)
	sort.SliceStable(highlightItems, func(i, j int) bool {
		return highlightItems[i].ImportanceScore > highlightItems[j].ImportanceScore
	})

	highlights := make([]string, 0, maxHighlights)
	for _, item := range highlightItems {
		if text := memorySummaryText(item); text != "" {
			highlights = append(highlights, text)
		}
		if len(highlights) == maxHighlights {
			break
		}
	}

	var memoryList strings.Builder
	for _, item := range items {
		memoryList.WriteString(fmt.Sprintf("- %s | %s | 重要度 %.1f",
			item.OccurredAt.Format("2006-01-02 15:04"),
			truncateSummaryText(memorySummaryText(item), maxMemoryListContent),
			item.ImportanceScore,
		))
		if mood := strings.TrimSpace(item.Mood); mood != "" {
			memoryList.WriteString(" | 情绪 " + mood)
		}
		memoryList.WriteString("\n")
	}

	return SummaryAggregation{
		Count:      len(items),
		Tags:       sortedCountKeys(tagCounts, maxAggregationValues),
		Moods:      sortedCountKeys(moodCounts, maxAggregationValues),
		Highlights: highlights,
		MemoryList: memoryList.String(),
	}
}

func parseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err == nil {
		return cleanTags(tags)
	}
	return cleanTags(strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	}))
}

func cleanTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag = strings.TrimSpace(tag); tag != "" {
			cleaned = append(cleaned, tag)
		}
	}
	return cleaned
}

func sortedCountKeys(counts map[string]int, limit int) []string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if counts[keys[i]] == counts[keys[j]] {
			return keys[i] < keys[j]
		}
		return counts[keys[i]] > counts[keys[j]]
	})
	if len(keys) > limit {
		keys = keys[:limit]
	}
	return keys
}

func memorySummaryText(item *model.MemoryItem) string {
	if item == nil {
		return ""
	}
	if summary := strings.TrimSpace(item.Summary); summary != "" {
		return summary
	}
	if content := strings.TrimSpace(item.ContentText); content != "" {
		return content
	}
	if imageURL := strings.TrimSpace(item.ImageURL); imageURL != "" {
		return "图片记忆：" + imageURL
	}
	return ""
}

func truncateSummaryText(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}
