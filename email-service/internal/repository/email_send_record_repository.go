package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
)

// EmailSendRecordRepository defines the interface for interacting with email send records.
type EmailSendRecordRepository interface {
	Create(record *model.EmailSendRecord) error
	Update(record *model.EmailSendRecord) error
	FindByID(id uint) (*model.EmailSendRecord, error)
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

func (r *emailSendRecordRepository) Update(record *model.EmailSendRecord) error {
	return r.db.Save(record).Error
}

func (r *emailSendRecordRepository) FindByID(id uint) (*model.EmailSendRecord, error) {
	var record model.EmailSendRecord
	err := r.db.First(&record, id).Error
	return &record, err
}
