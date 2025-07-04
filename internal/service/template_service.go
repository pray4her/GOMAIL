package service

import (
	"bytes"
	"fmt"
	"html/template"

	"email-service/internal/model"
	"email-service/internal/repository"
)

// PreviewData represents the sample data for rendering a template preview.
type PreviewData struct {
	Email     string                 `json:"email"`
	FirstName string                 `json:"first_name"`
	LastName  string                 `json:"last_name"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// RenderedTemplate represents the result of a template rendering.
type RenderedTemplate struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type TemplateService interface {
	CreateTemplate(template *model.EmailTemplate) error
	GetTemplates() ([]model.EmailTemplate, error)
	GetTemplateByID(id int64) (*model.EmailTemplate, error)
	UpdateTemplate(template *model.EmailTemplate) error
	DeleteTemplate(id int64) error
	RenderPreview(template *model.EmailTemplate, data PreviewData) (*RenderedTemplate, error)
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

func (s *templateService) RenderPreview(templateModel *model.EmailTemplate, data PreviewData) (*RenderedTemplate, error) {
	// Combine the data fields into a single map for templating
	templateData := make(map[string]interface{})
	if data.Metadata != nil {
		templateData = data.Metadata
	}
	templateData["Email"] = data.Email
	templateData["FirstName"] = data.FirstName
	templateData["LastName"] = data.LastName

	renderedSubject, renderedBody, err := renderTemplateWithData(templateModel.Subject, templateModel.Body, templateData)
	if err != nil {
		return nil, err
	}

	return &RenderedTemplate{
		Subject: renderedSubject,
		Body:    renderedBody,
	}, nil
}

// renderTemplateWithData is a reusable helper function for rendering templates.
func renderTemplateWithData(subjectTpl, bodyTpl string, data interface{}) (string, string, error) {
	// Render Subject
	subjectTmpl, err := template.New("subject").Parse(subjectTpl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse subject template: %w", err)
	}
	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	// Render Body
	bodyTmpl, err := template.New("body").Parse(bodyTpl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse body template: %w", err)
	}
	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute body template: %w", err)
	}

	return subjectBuf.String(), bodyBuf.String(), nil
}
