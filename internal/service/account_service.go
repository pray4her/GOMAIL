package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
)

type AccountService interface {
	CreateAccount(account *model.Account) error
	GetAccounts() ([]model.Account, error)
	GetAccountByID(id int64) (*model.Account, error)
	UpdateAccount(account *model.Account) error
	DeleteAccount(id int64) error
}

type accountService struct {
	repo repository.AccountRepository
}

func NewAccountService(repo repository.AccountRepository) AccountService {
	return &accountService{repo: repo}
}

func (s *accountService) CreateAccount(account *model.Account) error {
	// TODO: Encrypt AccessKeySecret before saving
	return s.repo.Create(account)
}

func (s *accountService) GetAccounts() ([]model.Account, error) {
	return s.repo.FindAll()
}

func (s *accountService) GetAccountByID(id int64) (*model.Account, error) {
	return s.repo.FindByID(id)
}

func (s *accountService) UpdateAccount(account *model.Account) error {
	// If secret is part of the update, it should be re-encrypted
	return s.repo.Update(account)
}

func (s *accountService) DeleteAccount(id int64) error {
	return s.repo.Delete(id)
}
