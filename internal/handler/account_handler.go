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
// @Description Request body for creating a new Aliyun email account.
type CreateAccountRequest struct {
	Name            string `json:"name" binding:"required" example:"Aliyun Account 1"`
	AccessKeyID     string `json:"access_key_id" binding:"required" example:"LTAI5txxxxxxxxxxxxxx"`
	AccessKeySecret string `json:"access_key_secret" binding:"required" example:"P1aXxxxxxxxxxxxxxxxxxxxxxx"`
	Domain          string `json:"domain" binding:"required" example:"mail.example.com"`
	DailySendLimit  int    `json:"daily_send_limit" example:"5000"`
}

type AccountHandler struct {
	service service.AccountService
}

func NewAccountHandler(service service.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// CreateAccount handles the creation of a new account.
// @Summary      Create Account
// @Description  Adds a new third-party email service provider account.
// @Tags         Accounts
// @Accept       json
// @Produce      json
// @Param        account  body      CreateAccountRequest  true  "Account Details"
// @Success      201      {object}  Response{data=model.Account}
// @Failure      400      {object}  Response
// @Failure      500      {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/accounts [post]
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
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
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to create account"})
		return
	}

	// The response `account` object will NOT contain the secret,
	// because the `json:"-"` tag on the model prevents it.
	c.JSON(http.StatusCreated, Response{Data: account})
}

// GetAccounts retrieves all accounts.
// @Summary      Get All Accounts
// @Description  Retrieves a list of all configured email service provider accounts.
// @Tags         Accounts
// @Produce      json
// @Success      200  {object}   Response{data=[]model.Account}
// @Failure      500  {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/accounts [get]
func (h *AccountHandler) GetAccounts(c *gin.Context) {
	accounts, err := h.service.GetAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to retrieve accounts"})
		return
	}
	c.JSON(http.StatusOK, Response{Data: accounts})
}

// GetAccount retrieves a single account by its ID.
// @Summary      Get Account by ID
// @Description  Retrieves the details of a specific account by its ID.
// @Tags         Accounts
// @Produce      json
// @Param        id   path      int  true  "Account ID"
// @Success      200  {object}  Response{data=model.Account}
// @Failure      400  {object}  Response  "Invalid account ID"
// @Failure      404  {object}  Response  "Account not found"
// @Failure      500  {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/accounts/{id} [get]
func (h *AccountHandler) GetAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid account ID"})
		return
	}

	account, err := h.service.GetAccountByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{Error: "Account not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to retrieve account"})
		return
	}

	c.JSON(http.StatusOK, Response{Data: account})
}

// UpdateAccount handles the update of an existing account.
// @Summary      Update Account
// @Description  Updates the details of an existing account. The request body should contain the full account object.
// @Tags         Accounts
// @Accept       json
// @Produce      json
// @Param        id       path      int            true  "Account ID"
// @Param        account  body      model.Account  true  "Account object with updated details"
// @Success      200      {object}  Response{data=model.Account}
// @Failure      400      {object}  Response  "Invalid account ID or request body"
// @Failure      500      {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/accounts/{id} [put]
func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid account ID"})
		return
	}

	var account model.Account
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}
	account.ID = id

	if err := h.service.UpdateAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to update account"})
		return
	}

	c.JSON(http.StatusOK, Response{Data: account})
}

// DeleteAccount handles the deletion of an account.
// @Summary      Delete Account
// @Description  Deletes an account by its ID.
// @Tags         Accounts
// @Produce      json
// @Param        id   path      int  true  "Account ID"
// @Success      200  {object}  Response{data=map[string]string} "message: Account deleted successfully"
// @Failure      400  {object}  Response "Invalid account ID"
// @Failure      500  {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/accounts/{id} [delete]
func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid account ID"})
		return
	}

	if err := h.service.DeleteAccount(id); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to delete account"})
		return
	}

	c.JSON(http.StatusOK, Response{Data: gin.H{"message": "Account deleted successfully"}})
}
