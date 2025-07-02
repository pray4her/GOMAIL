package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// RecipientService defines the interface for recipient-related business logic.
type RecipientService interface {
	CreateRecipient(email string, firstName, lastName *string, metadata map[string]interface{}) (*model.Recipient, error)
	GetRecipient(id uint) (*model.Recipient, error)
	ListRecipients(page, pageSize int) ([]model.Recipient, int64, error)
	UpdateRecipient(id uint, email *string, firstName, lastName *string, status *string, metadata map[string]interface{}) (*model.Recipient, error)
	DeleteRecipient(id uint) error
}

type recipientService struct {
	repo repository.RecipientRepository
}

// NewRecipientService creates a new RecipientService.
func NewRecipientService(repo repository.RecipientRepository) RecipientService {
	return &recipientService{repo: repo}
}

func (s *recipientService) CreateRecipient(email string, firstName, lastName *string, metadata map[string]interface{}) (*model.Recipient, error) {
	// Check if recipient already exists
	_, err := s.repo.FindByEmail(email)
	if err == nil {
		return nil, fmt.Errorf("recipient with email '%s' already exists", email)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check for existing recipient: %w", err)
	}

	// Marshal metadata if it exists
	var marshaledMetadata json.RawMessage
	if metadata != nil {
		var marshalErr error
		marshaledMetadata, marshalErr = json.Marshal(metadata)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", marshalErr)
		}
	}

	recipient := &model.Recipient{
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Status:    "subscribed",
		Metadata:  marshaledMetadata,
	}

	if err := s.repo.Create(recipient); err != nil {
		return nil, fmt.Errorf("failed to create recipient: %w", err)
	}
	return recipient, nil
}

func (s *recipientService) GetRecipient(id uint) (*model.Recipient, error) {
	return s.repo.FindByID(id)
}

func (s *recipientService) ListRecipients(page, pageSize int) ([]model.Recipient, int64, error) {
	return s.repo.List(page, pageSize)
}

func (s *recipientService) UpdateRecipient(id uint, email *string, firstName, lastName *string, status *string, metadata map[string]interface{}) (*model.Recipient, error) {
	recipient, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("recipient with id %d not found: %w", id, err)
	}

	if email != nil {
		// Check if the new email is already taken by another recipient
		existing, err := s.repo.FindByEmail(*email)
		if err == nil && existing.ID != int64(id) {
			return nil, fmt.Errorf("email '%s' is already in use by another recipient", *email)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check for existing email: %w", err)
		}
		recipient.Email = *email
	}
	if firstName != nil {
		recipient.FirstName = firstName
	}
	if lastName != nil {
		recipient.LastName = lastName
	}
	if status != nil {
		recipient.Status = *status
	}

	// Marshal and update metadata if it is provided
	if metadata != nil {
		marshaledMetadata, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		recipient.Metadata = marshaledMetadata
	}

	if err := s.repo.Update(recipient); err != nil {
		return nil, fmt.Errorf("failed to update recipient: %w", err)
	}
	return recipient, nil
}

func (s *recipientService) DeleteRecipient(id uint) error {
	// Check if recipient exists before deleting
	if _, err := s.repo.FindByID(id); err != nil {
		return fmt.Errorf("recipient with id %d not found: %w", id, err)
	}
	return s.repo.Delete(id)
}
