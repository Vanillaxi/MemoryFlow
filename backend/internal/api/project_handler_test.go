package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"memoryflow/internal/domain/repository"
	"memoryflow/internal/domain/service"

	"github.com/gin-gonic/gin"
)

func TestProjectHandlerCreateListAndGet(t *testing.T) {
	db, err := repository.InitSQLite(filepath.Join(t.TempDir(), "projects.db"))
	if err != nil {
		t.Fatal(err)
	}
	handler := NewProjectHandler(service.NewProjectService(repository.NewSQLiteProjectRepository(db)))
	router := gin.New()
	router.POST("/projects", handler.CreateProject)
	router.GET("/projects", handler.ListProjects)
	router.GET("/projects/:id", handler.GetProject)

	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewBufferString(`{"name":"MemoryFlow","repo_owner":"vanillaxi","repo_name":"MemoryFlow"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, request)
	if create.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}

	for _, path := range []string{"/projects", "/projects/1"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
		var payload map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
	}
}
