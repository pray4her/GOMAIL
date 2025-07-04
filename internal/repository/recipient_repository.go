package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
)

// RecipientRepository defines the interface for interacting with recipients.
type RecipientRepository interface {
	Create(recipient *model.Recipient) error
	Update(recipient *model.Recipient) error
	FindByID(id uint) (*model.Recipient, error)
	FindByIds(ids []uint) ([]model.Recipient, error)
	FindByEmail(email string) (*model.Recipient, error)
	List(page, pageSize int) ([]model.Recipient, int64, error)
	Delete(id uint) error
	FindByGroupID(groupID int64, page, pageSize int) ([]*model.Recipient, error)
}

type recipientRepository struct {
	db *gorm.DB
}

// NewRecipientRepository creates a new RecipientRepository.
func NewRecipientRepository(db *gorm.DB) RecipientRepository {
	return &recipientRepository{db: db}
}

func (r *recipientRepository) Create(recipient *model.Recipient) error {
	return r.db.Create(recipient).Error
}

func (r *recipientRepository) Update(recipient *model.Recipient) error {
	return r.db.Save(recipient).Error
}

func (r *recipientRepository) FindByID(id uint) (*model.Recipient, error) {
	var recipient model.Recipient
	err := r.db.First(&recipient, id).Error
	return &recipient, err
}

func (r *recipientRepository) FindByIds(ids []uint) ([]model.Recipient, error) {
	var recipients []model.Recipient
	err := r.db.Where("id IN ?", ids).Find(&recipients).Error
	return recipients, err
}

func (r *recipientRepository) FindByEmail(email string) (*model.Recipient, error) {
	var recipient model.Recipient
	err := r.db.Where("email = ?", email).First(&recipient).Error
	return &recipient, err
}

func (r *recipientRepository) List(page, pageSize int) ([]model.Recipient, int64, error) {
	var recipients []model.Recipient
	var total int64

	// Calculate offset
	offset := (page - 1) * pageSize

	// Query for total count
	if err := r.db.Model(&model.Recipient{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Query for paginated results
	err := r.db.Limit(pageSize).Offset(offset).Order("created_at desc").Find(&recipients).Error
	return recipients, total, err
}

func (r *recipientRepository) Delete(id uint) error {
	return r.db.Delete(&model.Recipient{}, id).Error
}

func (r *recipientRepository) FindByGroupID(groupID int64, page, pageSize int) ([]*model.Recipient, error) {
	var recipients []*model.Recipient
	offset := (page - 1) * pageSize
	err := r.db.
		Joins("JOIN recipient_group_members ON recipient_group_members.recipient_id = recipients.id").
		Where("recipient_group_members.group_id = ?", groupID).
		Limit(pageSize).
		Offset(offset).
		Order("recipients.id ASC").
		Find(&recipients).Error
	return recipients, err
}
