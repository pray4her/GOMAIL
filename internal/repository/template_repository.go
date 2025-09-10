package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
)

type TemplateRepository interface {
	Create(template *model.EmailTemplate) error
	FindAll() ([]model.EmailTemplate, error)
	List(page, pageSize int) ([]model.EmailTemplate, int64, error)
	FindByID(id int64) (*model.EmailTemplate, error)
	Update(template *model.EmailTemplate) error
	Delete(id int64) error
}

type templateRepository struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

func (r *templateRepository) Create(template *model.EmailTemplate) error {
	return r.db.Create(template).Error
}

func (r *templateRepository) FindAll() ([]model.EmailTemplate, error) {
	var templates []model.EmailTemplate
	err := r.db.Find(&templates).Error
	return templates, err
}

// List retrieves a paginated list of email templates.
func (r *templateRepository) List(page, pageSize int) ([]model.EmailTemplate, int64, error) {
	var templates []model.EmailTemplate
	var total int64

	// Get total count of templates
	if err := r.db.Model(&model.EmailTemplate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated templates
	offset := (page - 1) * pageSize
	err := r.db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&templates).Error
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

func (r *templateRepository) FindByID(id int64) (*model.EmailTemplate, error) {
	var template model.EmailTemplate
	err := r.db.First(&template, id).Error
	return &template, err
}

func (r *templateRepository) Update(template *model.EmailTemplate) error {
	return r.db.Save(template).Error
}

func (r *templateRepository) Delete(id int64) error {
	return r.db.Delete(&model.EmailTemplate{}, id).Error
}
