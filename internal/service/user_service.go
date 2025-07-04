package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
	"errors"
)

var (
	ErrUserAlreadyExists = errors.New("user with this username or email already exists")
)

// UserService defines the interface for user-related business logic.
type UserService interface {
	CreateUser(username, email, password string, isAdmin bool) (*model.User, error)
	GetUserByID(id int64) (*model.User, error)
	GetUserByUsername(username string) (*model.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new instance of UserService.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

// CreateUser handles the business logic of creating a new user.
func (s *userService) CreateUser(username, email, password string, isAdmin bool) (*model.User, error) {
	// Check for existing user
	if existingUser, _ := s.userRepo.FindByUsername(username); existingUser != nil {
		return nil, ErrUserAlreadyExists
	}
	// Note: In a real app, you'd also check for email uniqueness separately for a better UX.

	user := &model.User{
		Username: username,
		Email:    email,
		IsAdmin:  isAdmin,
		IsActive: true,
	}

	if err := user.SetPassword(password); err != nil {
		return nil, err
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID.
func (s *userService) GetUserByID(id int64) (*model.User, error) {
	return s.userRepo.FindByID(id)
}

// GetUserByUsername retrieves a user by their username.
func (s *userService) GetUserByUsername(username string) (*model.User, error) {
	return s.userRepo.FindByUsername(username)
}
