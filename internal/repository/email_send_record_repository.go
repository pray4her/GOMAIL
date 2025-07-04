package repository

import (
	"email-service/internal/model"
	"time"

	"gorm.io/gorm"
)

// EmailSendRecordRepository defines the interface for interacting with email send records.
type EmailSendRecordRepository interface {
	Create(record *model.EmailSendRecord) error
	CreateBatch(records []*model.EmailSendRecord) error
	Update(record *model.EmailSendRecord) error
	FindByID(id int64) (*model.EmailSendRecord, error)
	FindTrackableRecords(since time.Time, statuses []string) ([]*model.EmailSendRecord, error)
	FindByTaskID(taskID int64) ([]*model.EmailSendRecord, error)
	UpdateStatus(id int64, status, aliyunTaskID string, errorMessage *string) error
}

type emailSendRecordRepository struct {
	db *gorm.DB
}

// NewEmailSendRecordRepository creates a new EmailSendRecordRepository.
func NewEmailSendRecordRepository(db *gorm.DB) EmailSendRecordRepository {
	return &emailSendRecordRepository{db: db}
}

func (r *emailSendRecordRepository) Create(record *model.EmailSendRecord) error {
	return r.db.Create(record).Error
}

func (r *emailSendRecordRepository) CreateBatch(records []*model.EmailSendRecord) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.CreateInBatches(records, 100).Error
}

func (r *emailSendRecordRepository) Update(record *model.EmailSendRecord) error {
	return r.db.Save(record).Error
}

func (r *emailSendRecordRepository) FindByID(id int64) (*model.EmailSendRecord, error) {
	var record model.EmailSendRecord
	err := r.db.Preload("AccountSender.Account").Preload("AccountSender.Sender").First(&record, id).Error
	return &record, err
}

func (r *emailSendRecordRepository) FindTrackableRecords(since time.Time, statuses []string) ([]*model.EmailSendRecord, error) {
	var records []*model.EmailSendRecord
	err := r.db.Where("status IN ? AND aliyun_tag_name IS NOT NULL AND sent_at >= ?", statuses, since).Find(&records).Error
	return records, err
}

func (r *emailSendRecordRepository) FindByTaskID(taskID int64) ([]*model.EmailSendRecord, error) {
	var records []*model.EmailSendRecord
	err := r.db.
		Preload("Task").
		Preload("AccountSender.Account").
		Preload("AccountSender.Sender").
		Where("task_id = ?", taskID).
		Order("id asc").
		Find(&records).Error
	return records, err
}

// UpdateStatus updates specific fields of a send record to avoid GORM association magic.
func (r *emailSendRecordRepository) UpdateStatus(id int64, status, aliyunTaskID string, errorMessage *string) error {
	updates := map[string]interface{}{
		"status":                status,
		"last_status_update_at": time.Now(),
		"aliyun_task_id":        aliyunTaskID,
		"error_message":         errorMessage,
	}

	// Only set sent_at if the status is 'sent'
	if status == model.RecordStatusSent {
		updates["sent_at"] = time.Now()
	}

	return r.db.Model(&model.EmailSendRecord{}).Where("id = ?", id).Updates(updates).Error
}
