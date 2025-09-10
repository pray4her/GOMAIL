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
	FindByTaskIDs(taskIDs []int64) ([]*model.EmailSendRecord, error)
	UpdateStatus(recordID int64, status string, aliyunTaskID *string, errorMessage *string) error
	GetSentStatusCounts(taskID int64) (sent int64, failed int64, err error)
	GetAggregatedSentCounts(since time.Time) ([]*model.DailySenderSentCount, error)
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

func (r *emailSendRecordRepository) FindByTaskIDs(taskIDs []int64) ([]*model.EmailSendRecord, error) {
	var records []*model.EmailSendRecord
	if len(taskIDs) == 0 {
		return records, nil
	}
	// Explicitly select only the columns needed by the tracking service to avoid loading large data.
	err := r.db.Select("id", "task_id", "account_sender_id").Where("task_id IN ?", taskIDs).Find(&records).Error
	return records, err
}

// UpdateStatus updates the status and other tracking fields of a send record.
func (r *emailSendRecordRepository) UpdateStatus(recordID int64, status string, aliyunTaskID *string, errorMessage *string) error {
	updates := map[string]interface{}{
		"status":                status,
		"last_status_update_at": time.Now(),
	}
	if status == model.RecordStatusSent {
		updates["sent_at"] = time.Now()
	}

	if aliyunTaskID != nil {
		updates["aliyun_task_id"] = *aliyunTaskID
	}

	if errorMessage != nil {
		updates["error_message"] = errorMessage
	}

	return r.db.Model(&model.EmailSendRecord{}).Where("id = ?", recordID).Updates(updates).Error
}

// GetAggregatedSentCounts aggregates the count of successfully sent emails
// for each sender on each day since the provided timestamp.
func (r *emailSendRecordRepository) GetAggregatedSentCounts(since time.Time) ([]*model.DailySenderSentCount, error) {
	var results []*model.DailySenderSentCount

	err := r.db.Model(&model.EmailSendRecord{}).
		Select(`
			DATE(sent_at) as stat_date,
			account_sender_id,
			COUNT(*) as sent_count
		`).
		Where("status = ? AND sent_at >= ?", model.RecordStatusSent, since).
		Group("DATE(sent_at), account_sender_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}
	return results, nil
}

// GetSentStatusCounts counts the number of sent and failed records for a given task.
func (r *emailSendRecordRepository) GetSentStatusCounts(taskID int64) (sent int64, failed int64, err error) {
	var result struct {
		Sent   int64
		Failed int64
	}

	err = r.db.Model(&model.EmailSendRecord{}).
		Where("task_id = ? AND status = ?", taskID, model.RecordStatusSent).
		Count(&result.Sent).Error
	if err != nil {
		return 0, 0, err
	}

	err = r.db.Model(&model.EmailSendRecord{}).
		Where("task_id = ? AND status = ?", taskID, model.RecordStatusFailed).
		Count(&result.Failed).Error
	if err != nil {
		return 0, 0, err
	}

	return result.Sent, result.Failed, nil
}
