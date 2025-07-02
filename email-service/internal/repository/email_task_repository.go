package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
)

// EmailTaskRepository defines the interface for interacting with email tasks.
type EmailTaskRepository interface {
	Create(task *model.EmailTask) error
	FindByID(id uint) (*model.EmailTask, error)
}

type emailTaskRepository struct {
	db *gorm.DB
}

// NewEmailTaskRepository creates a new EmailTaskRepository.
func NewEmailTaskRepository(db *gorm.DB) EmailTaskRepository {
	return &emailTaskRepository{db: db}
}

// Create creates a new email task and its associations in a transaction.
func (r *emailTaskRepository) Create(task *model.EmailTask) error {
	return r.db.Create(task).Error
}

// FindByID finds an email task by its ID, preloading its recipients.
func (r *emailTaskRepository) FindByID(id uint) (*model.EmailTask, error) {
	var task model.EmailTask
	// Preload recipients when fetching a task
	err := r.db.Preload("Recipients").First(&task, id).Error
	return &task, err
}
