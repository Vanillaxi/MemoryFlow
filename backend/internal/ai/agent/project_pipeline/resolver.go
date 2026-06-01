package project_pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"memoryflow/internal/domain/model"
)

type ProjectLookup interface {
	GetProjectByID(ctx context.Context, id uint) (*model.Project, error)
	FindProjectFromMessage(ctx context.Context, message string) (*model.Project, error)
}

type ProjectResolver struct {
	projects ProjectLookup
}

func NewProjectResolver(projects ProjectLookup) *ProjectResolver {
	return &ProjectResolver{projects: projects}
}

func (r *ProjectResolver) Resolve(ctx context.Context, message string, projectID *uint) (*model.Project, error) {
	if r == nil || r.projects == nil {
		return nil, errors.New("project resolver is not initialized")
	}
	if projectID != nil {
		if *projectID == 0 {
			return nil, errors.New("project_id must be greater than zero")
		}
		project, err := r.projects.GetProjectByID(ctx, *projectID)
		if err != nil {
			return nil, fmt.Errorf("project_id %d not found: %w", *projectID, err)
		}
		return project, nil
	}
	project, err := r.projects.FindProjectFromMessage(ctx, strings.TrimSpace(message))
	if err != nil {
		return nil, fmt.Errorf("cannot resolve project from message; create the project first or specify project_id: %w", err)
	}
	return project, nil
}
