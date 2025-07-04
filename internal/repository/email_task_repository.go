package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
)

// EmailTaskRepository defines the interface for interacting with email tasks.
type EmailTaskRepository interface {
	Create(task *model.EmailTask) error
	FindByID(id int64) (*model.EmailTask, error)
	Update(task *model.EmailTask) error
	FindTrackableTasks(statuses []string) ([]*model.EmailTask, error)
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

// Update updates an existing email task.
func (r *emailTaskRepository) Update(task *model.EmailTask) error {
	return r.db.Save(task).Error
}

// FindTrackableTasks finds tasks that have a tracking tag and are in one of the given statuses.
func (r *emailTaskRepository) FindTrackableTasks(statuses []string) ([]*model.EmailTask, error) {
	var tasks []*model.EmailTask
	err := r.db.
		Where("aliyun_tag_name IS NOT NULL AND aliyun_tag_name != ''").
		Where("status IN (?)", statuses).
		Find(&tasks).Error
	return tasks, err
}

// FindByID finds an email task by its ID, preloading its sender details.
func (r *emailTaskRepository) FindByID(id int64) (*model.EmailTask, error) {
	var task model.EmailTask
	// The direct `Recipients` relation is deprecated in favor of RecipientGroupID.
	// The recipients will be resolved by the RecipientGroupService.
	err := r.db.
		Preload("AccountSender").
		First(&task, id).Error
	return &task, err
}
