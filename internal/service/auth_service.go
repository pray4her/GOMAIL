package service

import (
	"email-service/internal/config"
	"email-service/pkg/jwt"
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// AuthService defines the interface for authentication logic.
type AuthService interface {
	Login(username, password string) (string, error)
}

type authService struct {
	userService UserService
	jwtConfig   config.JWTConfig
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(userService UserService, jwtConfig config.JWTConfig) AuthService {
	return &authService{
		userService: userService,
		jwtConfig:   jwtConfig,
	}
}

// Login handles the user login process, validates credentials, and returns a JWT token.
func (s *authService) Login(username, password string) (string, error) {
	// Find user by username
	user, err := s.userService.GetUserByUsername(username)
	if err != nil {
		// Includes gorm.ErrRecordNotFound
		return "", ErrInvalidCredentials
	}

	// Check if the user is active
	if !user.IsActive {
		return "", ErrInvalidCredentials
	}

	// Check password
	if !user.CheckPassword(password) {
		return "", ErrInvalidCredentials
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, user.Username, s.jwtConfig.Secret, s.jwtConfig.ExpireHours)
	if err != nil {
		// This would be an internal server error
		return "", err
	}

	return token, nil
}
