package repository

import (
	"email-service/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AccountRepository interface {
	Create(account *model.Account) error
	FindAll() ([]model.Account, error)
	FindByID(id int64) (*model.Account, error)
	Update(account *model.Account) error
	Delete(id int64) error
	FindByIDForUpdate(id int64) (*model.Account, error)
}

type accountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

func (r *accountRepository) Create(account *model.Account) error {
	return r.db.Create(account).Error
}

func (r *accountRepository) FindAll() ([]model.Account, error) {
	var accounts []model.Account
	err := r.db.Find(&accounts).Error
	return accounts, err
}

func (r *accountRepository) FindByID(id int64) (*model.Account, error) {
	var account model.Account
	err := r.db.First(&account, id).Error
	return &account, err
}

func (r *accountRepository) Update(account *model.Account) error {
	return r.db.Save(account).Error
}

func (r *accountRepository) Delete(id int64) error {
	return r.db.Delete(&model.Account{}, id).Error
}

func (r *accountRepository) FindByIDForUpdate(id int64) (*model.Account, error) {
	var account model.Account
	err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, id).Error
	return &account, err
}
