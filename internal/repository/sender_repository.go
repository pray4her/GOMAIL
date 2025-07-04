package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
)

type SenderRepository interface {
	// Sender operations
	CreateSender(sender *model.Sender) error
	FindSenderByID(id int64) (*model.Sender, error)

	// Account-Sender association operations
	CreateAccountSender(accountSender *model.AccountSender) error
	FindAccountSenderByID(id int64) (*model.AccountSender, error)
	FindSendersByAccountID(accountID int64) ([]model.AccountSender, error)
	FindAccountSenderDetails(accountSenderID int64) (*model.AccountSender, error)
	FindAccountSenderDetailsByIDs(ids []int64) ([]model.AccountSender, error)
}

type senderRepository struct {
	db *gorm.DB
}

func NewSenderRepository(db *gorm.DB) SenderRepository {
	return &senderRepository{db: db}
}

// --- Sender operations ---

func (r *senderRepository) CreateSender(sender *model.Sender) error {
	return r.db.Create(sender).Error
}

func (r *senderRepository) FindSenderByID(id int64) (*model.Sender, error) {
	var sender model.Sender
	err := r.db.First(&sender, id).Error
	return &sender, err
}

// --- Account-Sender association operations ---

func (r *senderRepository) CreateAccountSender(accountSender *model.AccountSender) error {
	return r.db.Create(accountSender).Error
}

func (r *senderRepository) FindAccountSenderByID(id int64) (*model.AccountSender, error) {
	var accountSender model.AccountSender
	err := r.db.First(&accountSender, id).Error
	return &accountSender, err
}

func (r *senderRepository) FindSendersByAccountID(accountID int64) ([]model.AccountSender, error) {
	var accountSenders []model.AccountSender
	err := r.db.Preload("Sender").Where("account_id = ?", accountID).Find(&accountSenders).Error
	return accountSenders, err
}

// FindAccountSenderDetails uses JOINs to fetch Account and Sender details in a single query.
func (r *senderRepository) FindAccountSenderDetails(accountSenderID int64) (*model.AccountSender, error) {
	var accountSender model.AccountSender
	err := r.db.
		Joins("Account").
		Joins("Sender").
		First(&accountSender, accountSenderID).Error
	return &accountSender, err
}

// FindAccountSenderDetailsByIDs efficiently fetches multiple AccountSenders and their
// associated Account and Sender details in a minimal number of queries.
func (r *senderRepository) FindAccountSenderDetailsByIDs(ids []int64) ([]model.AccountSender, error) {
	var accountSenders []model.AccountSender
	if len(ids) == 0 {
		return accountSenders, nil
	}
	err := r.db.
		Preload("Account").
		Preload("Sender").
		Where("id IN ?", ids).
		Find(&accountSenders).Error
	return accountSenders, err
}
