package project_pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"memoryflow/internal/domain/model"
)

type fakeProjectLookup struct {
	byID        *model.Project
	fromMessage *model.Project
}

func (f fakeProjectLookup) GetProjectByID(context.Context, uint) (*model.Project, error) {
	if f.byID == nil {
		return nil, errors.New("not found")
	}
	return f.byID, nil
}

func (f fakeProjectLookup) FindProjectFromMessage(context.Context, string) (*model.Project, error) {
	if f.fromMessage == nil {
		return nil, errors.New("not found")
	}
	return f.fromMessage, nil
}

func TestProjectResolverUsesProjectID(t *testing.T) {
	id := uint(1)
	project, err := NewProjectResolver(fakeProjectLookup{byID: &model.Project{ID: id, Name: "MemoryFlow"}}).Resolve(context.Background(), "ignored", &id)
	if err != nil || project.ID != id {
		t.Fatalf("project=%#v err=%v", project, err)
	}
}

func TestProjectResolverUsesMessage(t *testing.T) {
	project, err := NewProjectResolver(fakeProjectLookup{fromMessage: &model.Project{Name: "MemoryFlow"}}).Resolve(context.Background(), "MemoryFlow 最近做到哪了", nil)
	if err != nil || project.Name != "MemoryFlow" {
		t.Fatalf("project=%#v err=%v", project, err)
	}
}

func TestProjectResolverReportsMissingProject(t *testing.T) {
	_, err := NewProjectResolver(fakeProjectLookup{}).Resolve(context.Background(), "未知项目", nil)
	if err == nil || !strings.Contains(err.Error(), "create the project first or specify project_id") {
		t.Fatalf("unexpected err: %v", err)
	}
}
