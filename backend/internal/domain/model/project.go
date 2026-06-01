package model

import "time"

type Project struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null;uniqueIndex" json:"name"`
	Description string    `json:"description"`
	RepoOwner   string    `gorm:"not null" json:"repo_owner"`
	RepoName    string    `gorm:"not null" json:"repo_name"`
	RepoURL     string    `json:"repo_url"`
	TechStack   string    `json:"tech_stack"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p Project) Repository() string {
	return p.RepoOwner + "/" + p.RepoName
}
