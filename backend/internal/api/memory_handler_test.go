package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"memoryflow/internal/ai/agent/chat_pipeline"

	"github.com/gin-gonic/gin"
)

func TestListAgentToolsOnlyReturnsExternalCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := &MemoryHandler{chatPipeline: &chat_pipeline.Pipeline{}}
	router.GET("/api/agent/tools", handler.ListAgentTools)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/agent/tools", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}

	want := []string{"get_current_time", "query_long_term_memory", "get_memory_detail", "aggregate_memory"}
	if len(response.Data) != len(want) {
		t.Fatalf("len(tools) = %d, want %d: %s", len(response.Data), len(want), recorder.Body.String())
	}
	for i := range want {
		if response.Data[i].Name != want[i] {
			t.Fatalf("tool[%d] = %q, want %q", i, response.Data[i].Name, want[i])
		}
	}
}
