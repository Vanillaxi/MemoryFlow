package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"memoryflow/internal/domain/model"
	"memoryflow/internal/domain/repository"

	"gorm.io/gorm"
)

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RepoOwner   string `json:"repo_owner"`
	RepoName    string `json:"repo_name"`
	RepoURL     string `json:"repo_url"`
	TechStack   string `json:"tech_stack"`
	Status      string `json:"status"`
}

type ProjectService struct {
	repo repository.ProjectRepository
}

func NewProjectService(repo repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) CreateProject(ctx context.Context, req CreateProjectRequest) (*model.Project, error) {
	project := &model.Project{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		RepoOwner:   strings.TrimSpace(req.RepoOwner),
		RepoName:    strings.TrimSpace(req.RepoName),
		RepoURL:     strings.TrimSpace(req.RepoURL),
		TechStack:   strings.TrimSpace(req.TechStack),
		Status:      strings.TrimSpace(req.Status),
	}
	if project.Name == "" || project.RepoOwner == "" || project.RepoName == "" {
		return nil, errors.New("name, repo_owner and repo_name are required")
	}
	if err := s.repo.Create(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) ListProjects(ctx context.Context) ([]model.Project, error) {
	return s.repo.List(ctx)
}

func (s *ProjectService) GetProjectByID(ctx context.Context, id uint) (*model.Project, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProjectService) FindProjectByName(ctx context.Context, name string) (*model.Project, error) {
	return s.repo.FindByName(ctx, strings.TrimSpace(name))
}

func (s *ProjectService) FindProjectFromMessage(ctx context.Context, message string) (*model.Project, error) {
	projects, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	normalized := strings.ToLower(message)
	for i := range projects {
		if strings.Contains(normalized, strings.ToLower(projects[i].Name)) {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project not found in message: %w", gorm.ErrRecordNotFound)
}
