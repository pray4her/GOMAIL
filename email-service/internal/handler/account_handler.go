package handler

import (
	"email-service/internal/model"
	"email-service/internal/service"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAccountRequest defines the request body for creating an account.
type CreateAccountRequest struct {
	Name            string `json:"name" binding:"required"`
	AccessKeyID     string `json:"access_key_id" binding:"required"`
	AccessKeySecret string `json:"access_key_secret" binding:"required"`
	Domain          string `json:"domain" binding:"required"`
	DailySendLimit  int    `json:"daily_send_limit"`
}

type AccountHandler struct {
	service service.AccountService
}

func NewAccountHandler(service service.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// CreateAccount handles the creation of a new account.
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply default value if not provided
	if req.DailySendLimit == 0 {
		req.DailySendLimit = 2000
	}

	// Map request DTO to the database model
	account := &model.Account{
		Name:            req.Name,
		AccessKeyID:     req.AccessKeyID,
		AccessKeySecret: req.AccessKeySecret,
		Domain:          req.Domain,
		DailySendLimit:  req.DailySendLimit,
	}

	if err := h.service.CreateAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}

	// The response `account` object will NOT contain the secret,
	// because the `json:"-"` tag on the model prevents it.
	c.JSON(http.StatusCreated, account)
}

// GetAccounts retrieves all accounts.
func (h *AccountHandler) GetAccounts(c *gin.Context) {
	accounts, err := h.service.GetAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve accounts"})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

// GetAccount retrieves a single account by its ID.
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	account, err := h.service.GetAccountByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve account"})
		return
	}

	c.JSON(http.StatusOK, account)
}

// UpdateAccount handles the update of an existing account.
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	var account model.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account.ID = id

	if err := h.service.UpdateAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account"})
		return
	}

	c.JSON(http.StatusOK, account)
}

// DeleteAccount handles the deletion of an account.
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	if err := h.service.DeleteAccount(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account deleted successfully"})
}
