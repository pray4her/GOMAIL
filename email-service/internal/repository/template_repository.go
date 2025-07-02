package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
)

type TemplateRepository interface {
	Create(template *model.EmailTemplate) error
	FindAll() ([]model.EmailTemplate, error)
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
