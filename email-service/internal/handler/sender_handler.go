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
func (h *SenderHandler) CreateSender(c *gin.Context) {
	var sender model.Sender
	if err := c.ShouldBindJSON(&sender); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.CreateSender(&sender); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create sender"})
		return
	}

	c.JSON(http.StatusCreated, sender)
}

// AddSenderToAccountRequest defines the request body for associating a sender with an account.
type AddSenderToAccountRequest struct {
	EmailAddress   string `json:"email_address" binding:"required,email"`
	DailySendLimit int    `json:"daily_send_limit" binding:"required,gt=0"`
}

// AddSenderToAccount handles associating a sender with an account.
func (h *SenderHandler) AddSenderToAccount(c *gin.Context) {
	accountID, err := strconv.ParseInt(c.Param("accountId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return
	}

	senderID, err := strconv.ParseInt(c.Param("senderId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sender ID"})
		return
	}

	var req AddSenderToAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountSender, err := h.service.AddSenderToAccount(accountID, senderID, req.EmailAddress, req.DailySendLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, accountSender)
}
