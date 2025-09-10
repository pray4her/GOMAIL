package repository

import (
	"email-service/internal/model"
	"time"

	"gorm.io/gorm"
)

// RecipientImportTaskRepository defines the interface for interacting with recipient import tasks.
type RecipientImportTaskRepository interface {
	Create(task *model.RecipientImportTask) error
	FindByID(id int64) (*model.RecipientImportTask, error)
	FindByUserID(userID int64, page, pageSize int) ([]*model.RecipientImportTask, int64, error)
	FindInProgressTasks() ([]*model.RecipientImportTask, error)
	UpdateStatus(id int64, status string, errorMessage *string) error
	UpdateProgress(id int64, processed, success, failed int, total int) error
	UpdateResult(id int64, status string, completedAt *time.Time) error
	UpdateStartTime(id int64, startedAt time.Time) error
}

type recipientImportTaskRepository struct {
	db *gorm.DB
}

// NewRecipientImportTaskRepository creates a new RecipientImportTaskRepository.
func NewRecipientImportTaskRepository(db *gorm.DB) RecipientImportTaskRepository {
	return &recipientImportTaskRepository{db: db}
}

func (r *recipientImportTaskRepository) Create(task *model.RecipientImportTask) error {
	return r.db.Create(task).Error
}

func (r *recipientImportTaskRepository) FindByID(id int64) (*model.RecipientImportTask, error) {
	var task model.RecipientImportTask
	err := r.db.Preload("CreatedByUser").First(&task, id).Error
	return &task, err
}

func (r *recipientImportTaskRepository) FindByUserID(userID int64, page, pageSize int) ([]*model.RecipientImportTask, int64, error) {
	var tasks []*model.RecipientImportTask
	var total int64

	// Count total records
	if err := r.db.Model(&model.RecipientImportTask{}).Where("created_by_user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	err := r.db.Where("created_by_user_id = ?", userID).
		Preload("CreatedByUser").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&tasks).Error

	return tasks, total, err
}

func (r *recipientImportTaskRepository) FindInProgressTasks() ([]*model.RecipientImportTask, error) {
	var tasks []*model.RecipientImportTask
	err := r.db.Where("status = ?", model.ImportTaskStatusProcessing).
		Preload("CreatedByUser").
		Order("created_at ASC").
		Find(&tasks).Error
	return tasks, err
}

func (r *recipientImportTaskRepository) UpdateStatus(id int64, status string, errorMessage *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if errorMessage != nil {
		updates["error_message"] = *errorMessage
	}
	return r.db.Model(&model.RecipientImportTask{}).Where("id = ?", id).Updates(updates).Error
}

func (r *recipientImportTaskRepository) UpdateProgress(id int64, processed, success, failed, total int) error {
	updates := map[string]interface{}{
		"processed_records": processed,
		"success_records":   success,
		"failed_records":    failed,
		"total_records":     total,
		"updated_at":        time.Now(),
	}
	return r.db.Model(&model.RecipientImportTask{}).Where("id = ?", id).Updates(updates).Error
}

func (r *recipientImportTaskRepository) UpdateResult(id int64, status string, completedAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if completedAt != nil {
		updates["completed_at"] = *completedAt
	}

	return r.db.Model(&model.RecipientImportTask{}).Where("id = ?", id).Updates(updates).Error
}

func (r *recipientImportTaskRepository) UpdateStartTime(id int64, startedAt time.Time) error {
	updates := map[string]interface{}{
		"started_at": startedAt,
		"updated_at": time.Now(),
	}
	return r.db.Model(&model.RecipientImportTask{}).Where("id = ?", id).Updates(updates).Error
}
