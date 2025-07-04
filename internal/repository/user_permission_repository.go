package repository

import (
	"errors"

	"gorm.io/gorm"
)

// UserPermission is a placeholder for the model, as it's a simple join table.
// We interact with it directly via the repository.
type UserPermission struct {
	UserID          int64 `gorm:"primaryKey"`
	AccountSenderID int64 `gorm:"primaryKey"`
}

// TableName sets the table name for the UserPermission model.
func (UserPermission) TableName() string {
	return "user_permissions"
}

// UserPermissionRepository defines the interface for user permission data operations.
type UserPermissionRepository interface {
	HasPermission(userID, accountSenderID int64) (bool, error)
	GrantPermission(userID, accountSenderID int64) error
	RevokePermission(userID, accountSenderID int64) error
	FindAllowedAccountSenderIDs(userID int64) ([]int64, error)
}

type gormUserPermissionRepository struct {
	db *gorm.DB
}

// NewGORMUserPermissionRepository creates a new instance of UserPermissionRepository.
func NewGORMUserPermissionRepository(db *gorm.DB) UserPermissionRepository {
	return &gormUserPermissionRepository{db: db}
}

// HasPermission checks if a user has permission to use a specific account sender.
func (r *gormUserPermissionRepository) HasPermission(userID, accountSenderID int64) (bool, error) {
	var count int64
	err := r.db.Model(&UserPermission{}).Where("user_id = ? AND account_sender_id = ?", userID, accountSenderID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GrantPermission gives a user permission to use an account sender.
func (r *gormUserPermissionRepository) GrantPermission(userID, accountSenderID int64) error {
	permission := UserPermission{
		UserID:          userID,
		AccountSenderID: accountSenderID,
	}
	// Using FirstOrCreate to avoid duplicate entries.
	return r.db.FirstOrCreate(&permission).Error
}

// RevokePermission removes a user's permission to use an account sender.
func (r *gormUserPermissionRepository) RevokePermission(userID, accountSenderID int64) error {
	result := r.db.Where("user_id = ? AND account_sender_id = ?", userID, accountSenderID).Delete(&UserPermission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("permission not found")
	}
	return nil
}

// FindAllowedAccountSenderIDs returns a slice of account sender IDs that the user has permission to use.
func (r *gormUserPermissionRepository) FindAllowedAccountSenderIDs(userID int64) ([]int64, error) {
	var ids []int64
	err := r.db.Model(&UserPermission{}).Where("user_id = ?", userID).Pluck("account_sender_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
