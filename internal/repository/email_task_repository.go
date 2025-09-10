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
	UpdateStatus(taskID int64, status string) error
	UpdateProgress(taskID int64, sentCount, failedCount int) error
	FindInProgressTasks() ([]*model.EmailTask, error)
	FindTrackableTasks(statuses []string) ([]*model.EmailTask, error)
	List(page, pageSize int) ([]model.EmailTask, int64, error)
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
	// We no longer preload AccountSender as it's determined dynamically by the LoadBalancer.
	err := r.db.
		First(&task, id).Error
	return &task, err
}

// List retrieves a paginated list of email tasks.
func (r *emailTaskRepository) List(page, pageSize int) ([]model.EmailTask, int64, error) {
	var tasks []model.EmailTask
	var total int64

	// Get total count of tasks
	if err := r.db.Model(&model.EmailTask{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated tasks
	offset := (page - 1) * pageSize
	err := r.db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// UpdateStatus updates only the status field of an email task.
func (r *emailTaskRepository) UpdateStatus(taskID int64, status string) error {
	return r.db.Model(&model.EmailTask{}).Where("id = ?", taskID).Update("status", status).Error
}

// UpdateProgress updates the progress tracking fields of an email task.
func (r *emailTaskRepository) UpdateProgress(taskID int64, sentCount, failedCount int) error {
	updates := map[string]interface{}{
		"sent_count":   sentCount,
		"failed_count": failedCount,
	}
	return r.db.Model(&model.EmailTask{}).Where("id = ?", taskID).Updates(updates).Error
}

// FindInProgressTasks finds tasks that are currently being processed.
func (r *emailTaskRepository) FindInProgressTasks() ([]*model.EmailTask, error) {
	var tasks []*model.EmailTask
	inProgressStatuses := []string{
		model.TaskStatusDispatching,
		model.TaskStatusProcessing,
	}
	err := r.db.Where("status IN (?)", inProgressStatuses).Find(&tasks).Error
	return tasks, err
}
