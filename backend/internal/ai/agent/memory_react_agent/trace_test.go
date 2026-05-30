package memory_react_agent

import (
	"errors"
	"sync"
	"testing"
)

func TestTraceCollectorLifecycle(t *testing.T) {
	collector := NewTraceCollector("test_mode")
	collector.Start("node", map[string]any{"input": "value"})
	collector.End("node", map[string]any{"output": "value"})
	collector.Error("node", errors.New("boom"))

	trace := collector.Trace()
	if trace.Mode != "test_mode" {
		t.Fatalf("Mode = %q, want test_mode", trace.Mode)
	}
	if len(trace.Steps) != 3 {
		t.Fatalf("len(Steps) = %d, want 3", len(trace.Steps))
	}
	if trace.Steps[0].Event != "start" || trace.Steps[1].Event != "end" || trace.Steps[2].Event != "error" {
		t.Fatalf("unexpected events: %#v", trace.Steps)
	}
	if trace.Error != "boom" {
		t.Fatalf("Error = %q, want boom", trace.Error)
	}
}

func TestTraceCollectorConcurrentAppend(t *testing.T) {
	collector := NewTraceCollector("concurrent")
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.Start("node", nil)
			collector.End("node", nil)
		}()
	}
	wg.Wait()

	if got := len(collector.Trace().Steps); got != 100 {
		t.Fatalf("len(Steps) = %d, want 100", got)
	}
}

func TestSanitizeTraceJSON(t *testing.T) {
	got := sanitizeTraceJSON(`{"api_key":"key","authorization":"bearer","nested":{"token":"value","secret":"hidden"},"ok":"visible"}`)
	want := `{"api_key":"[redacted]","authorization":"[redacted]","nested":{"secret":"[redacted]","token":"[redacted]"},"ok":"visible"}`
	if got != want {
		t.Fatalf("sanitizeTraceJSON() = %s, want %s", got, want)
	}
}
