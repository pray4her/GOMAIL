package handler

import (
	"email-service/internal/model"
	"email-service/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SenderHandler struct {
	service service.SenderService
}

func NewSenderHandler(service service.SenderService) *SenderHandler {
	return &SenderHandler{service: service}
}

// CreateSender handles the creation of a new sender.
// @Summary      Create Sender
// @Description  Creates a new sender entity (e.g., a person or department).
// @Tags         Senders
// @Accept       json
// @Produce      json
// @Param        sender  body      model.Sender  true  "Sender Information"
// @Success      201     {object}  Response{data=model.Sender}
// @Failure      400     {object}  Response
// @Failure      500     {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/senders [post]
func (h *SenderHandler) CreateSender(c *gin.Context) {
	var sender model.Sender
	if err := c.ShouldBindJSON(&sender); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	if err := h.service.CreateSender(&sender); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to create sender"})
		return
	}

	c.JSON(http.StatusCreated, Response{Data: sender})
}

// AddSenderToAccountRequest defines the request body for associating a sender with an account.
type AddSenderToAccountRequest struct {
	EmailAddress   string `json:"email_address" binding:"required,email" example:"noreply@example.com"`
	DailySendLimit int    `json:"daily_send_limit" binding:"required,gt=0" example:"1000"`
}

// AddSenderToAccount handles associating a sender with an account.
// @Summary      Add Sender to Account
// @Description  Associates a sender with an account, creating a usable 'from' email address with a specific sending limit.
// @Tags         Senders
// @Accept       json
// @Produce      json
// @Param        accountId  path      int                        true  "Account ID"
// @Param        senderId   path      int                        true  "Sender ID"
// @Param        payload    body      AddSenderToAccountRequest  true  "Association Details"
// @Success      201        {object}  Response{data=model.AccountSender}
// @Failure      400        {object}  Response  "Invalid ID or request body"
// @Failure      500        {object}  Response  "Failed to associate sender with account"
// @Security     ApiKeyAuth
// @Router       /api/v1/senders/{senderId}/accounts/{accountId} [post]
func (h *SenderHandler) AddSenderToAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("accountId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid account ID"})
		return
	}

	senderID, err := strconv.ParseInt(c.Param("senderId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid sender ID"})
		return
	}

	var req AddSenderToAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	accountSender, err := h.service.AddSenderToAccount(accountID, senderID, req.EmailAddress, req.DailySendLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, Response{Data: accountSender})
}

// GetSendersByAccountID handles retrieving all senders associated with a specific account.
// @Summary Get Senders by Account ID
// @Description Retrieves a paginated list of all senders (AccountSender) associated with a given account ID.
// @Tags Senders
// @Produce json
// @Param id path int true "Account ID"
// @Param page query int false "Page number for pagination" default(1)
// @Param page_size query int false "Number of items per page" default(10)
// @Success 200 {object} Response{data=model.PaginatedAccountSenders} "A paginated list of account senders"
// @Failure 400 {object} Response "Invalid ID or query parameters"
// @Failure 500 {object} Response "Failed to retrieve senders"
// @Security ApiKeyAuth
// @Router /api/v1/accounts/{id}/senders [get]
func (h *SenderHandler) GetSendersByAccountID(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid account ID"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize <= 0 {
		pageSize = 10
	}

	paginatedSenders, err := h.service.GetSendersByAccountID(accountID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Data: paginatedSenders})
}
