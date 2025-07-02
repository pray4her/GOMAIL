package handler

import (
	"context"
	"email-service/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// EmailService defines the interface for email-related business logic.
// It's a subset of the actual EmailService to promote loose coupling.
type EmailService interface {
	QueueEmailForSending(ctx context.Context, accountSenderID int64, toAddress, subject, htmlBody string) (*model.EmailSendRecord, error)
}

// EmailHandler handles HTTP requests for emails.
type EmailHandler struct {
	service EmailService
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler(service EmailService) *EmailHandler {
	return &EmailHandler{service: service}
}

// SendSingleEmailRequest defines the request body for sending a single email.
type SendSingleEmailRequest struct {
	AccountSenderID int64  `json:"account_sender_id" binding:"required"`
	ToAddress       string `json:"to_address" binding:"required,email"`
	Subject         string `json:"subject" binding:"required"`
	HtmlBody        string `json:"html_body" binding:"required"`
}

// SendSingleEmail handles the API endpoint for sending a single email.
// It now queues the email for asynchronous sending.
func (h *EmailHandler) SendSingleEmail(c *gin.Context) {
	var req SendSingleEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use the request context for the service call.
	ctx := c.Request.Context()

	record, err := h.service.QueueEmailForSending(ctx, req.AccountSenderID, req.ToAddress, req.Subject, req.HtmlBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Email has been accepted for sending.",
		"task_id": record.ID,
	})
}
