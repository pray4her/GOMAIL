package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
)

type TemplateService interface {
	CreateTemplate(template *model.EmailTemplate) error
	GetTemplates() ([]model.EmailTemplate, error)
	GetTemplateByID(id int64) (*model.EmailTemplate, error)
	UpdateTemplate(template *model.EmailTemplate) error
	DeleteTemplate(id int64) error
}

type templateService struct {
	repo repository.TemplateRepository
}

func NewTemplateService(repo repository.TemplateRepository) TemplateService {
	return &templateService{repo: repo}
}

func (s *templateService) CreateTemplate(template *model.EmailTemplate) error {
	return s.repo.Create(template)
}

func (s *templateService) GetTemplates() ([]model.EmailTemplate, error) {
	return s.repo.FindAll()
}

func (s *templateService) GetTemplateByID(id int64) (*model.EmailTemplate, error) {
	return s.repo.FindByID(id)
}

func (s *templateService) UpdateTemplate(template *model.EmailTemplate) error {
	return s.repo.Update(template)
}

func (s *templateService) DeleteTemplate(id int64) error {
	return s.repo.Delete(id)
}
