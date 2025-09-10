package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
	"fmt"
)

type SenderService interface {
	CreateSender(sender *model.Sender) error
	AddSenderToAccount(accountID, senderID int64, emailAddress string, dailySendLimit int) (*model.AccountSender, error)
	GetSendersByAccountID(accountID int64, page, pageSize int) (*model.PaginatedAccountSenders, error)
}

type senderService struct {
	senderRepo  repository.SenderRepository
	accountRepo repository.AccountRepository
}

func NewSenderService(senderRepo repository.SenderRepository, accountRepo repository.AccountRepository) SenderService {
	return &senderService{
		senderRepo:  senderRepo,
		accountRepo: accountRepo,
	}
}

func (s *senderService) CreateSender(sender *model.Sender) error {
	return s.senderRepo.CreateSender(sender)
}

func (s *senderService) AddSenderToAccount(accountID, senderID int64, emailAddress string, dailySendLimit int) (*model.AccountSender, error) {
	// 1. Verify that the account exists
	_, err := s.accountRepo.FindByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("account with id %d not found: %w", accountID, err)
	}

	// 2. Verify that the sender exists
	_, err = s.senderRepo.FindSenderByID(senderID)
	if err != nil {
		return nil, fmt.Errorf("sender with id %d not found: %w", senderID, err)
	}

	// 3. Create the association
	accountSender := &model.AccountSender{
		AccountID:      accountID,
		SenderID:       senderID,
		EmailAddress:   emailAddress,
		DailySendLimit: dailySendLimit,
		// Status and Weight have default values in the model
	}

	err = s.senderRepo.CreateAccountSender(accountSender)
	if err != nil {
		return nil, fmt.Errorf("failed to add sender to account: %w", err)
	}

	return accountSender, nil
}

func (s *senderService) GetSendersByAccountID(accountID int64, page, pageSize int) (*model.PaginatedAccountSenders, error) {
	return s.senderRepo.FindSendersByAccountID(accountID, page, pageSize)
}
