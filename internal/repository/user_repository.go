package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	Create(user *model.User) error
	FindByUsername(username string) (*model.User, error)
	FindByID(id int64) (*model.User, error)
}

// gormUserRepository is an implementation of UserRepository using GORM.
type gormUserRepository struct {
	db *gorm.DB
}

// NewGORMUserRepository creates a new instance of UserRepository.
func NewGORMUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

// Create creates a new user record in the database.
func (r *gormUserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// FindByUsername finds a user by their username.
func (r *gormUserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by their ID.
func (r *gormUserRepository) FindByID(id int64) (*model.User, error) {
	var user model.User
	err := r.db.Preload(clause.Associations).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
