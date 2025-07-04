package handler

import (
	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// EmailHandler handles HTTP requests related to sending emails.
type EmailHandler struct {
	senderRepo repository.SenderRepository
	recordRepo repository.EmailSendRecordRepository
	queue      queue.QueueService
}

// NewEmailHandler creates a new EmailHandler.
func NewEmailHandler(senderRepo repository.SenderRepository, recordRepo repository.EmailSendRecordRepository, queue queue.QueueService) *EmailHandler {
	return &EmailHandler{
		senderRepo: senderRepo,
		recordRepo: recordRepo,
		queue:      queue,
	}
}

type SendEmailRequest struct {
	AccountSenderID int64  `json:"account_sender_id" binding:"required" example:"1"`
	ToAddress       string `json:"to_address" binding:"required,email" example:"test.recipient@example.com"`
	Subject         string `json:"subject" binding:"required" example:"A Quick Message"`
	HTMLBody        string `json:"html_body" binding:"required" example:"<p>Hello, this is a test email.</p>"`
}

// SendSingleEmail queues a single email for sending.
// @Summary      Send Single Email
// @Description  Queues a single, non-template email for immediate sending. This is suitable for transactional emails.
// @Tags         Emails
// @Accept       json
// @Produce      json
// @Param        email   body      SendEmailRequest  true  "Email Details"
// @Success      202     {object}  Response{data=map[string]interface{}} "message: Email queued for sending, record_id: 12345"
// @Failure      400     {object}  Response  "Invalid request body"
// @Failure      500     {object}  Response  "Internal server error"
// @Security     ApiKeyAuth
// @Router       /api/v1/emails/send [post]
func (h *EmailHandler) SendSingleEmail(c *gin.Context) {
	var req SendEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	// For single sends, we replicate the logic from the TaskDispatcher to avoid N+1 in workers.
	// 1. Fetch complete sender details for credentials.
	accountSender, err := h.senderRepo.FindAccountSenderDetails(req.AccountSenderID)
	if err != nil {
		// This could be a 404 Not Found if the ID is invalid.
		c.JSON(http.StatusNotFound, Response{Error: fmt.Sprintf("Account sender with ID %d not found or invalid: %v", req.AccountSenderID, err)})
		return
	}

	// 2. Create the initial send record with "pending" status.
	// For single sends, TaskID is nil.
	record := &model.EmailSendRecord{
		TaskID:          nil,
		AccountSenderID: req.AccountSenderID,
		RecipientEmail:  req.ToAddress,
		Subject:         req.Subject,
		Body:            req.HTMLBody,
		Status:          model.RecordStatusPending,
	}
	if err := h.recordRepo.Create(record); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to create send record: " + err.Error()})
		return
	}

	// 3. Create the job payload with all necessary data.
	// For single sends, there is no AliyunTagName.
	payload := model.EmailJobPayload{
		RecordID:       record.ID,
		RecipientEmail: record.RecipientEmail,
		Subject:        record.Subject,
		Body:           record.Body,
		AccountSender:  *accountSender,
		AliyunTagName:  "", // No tag for single ad-hoc emails
	}

	// 4. Marshal and enqueue the payload.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to create email job payload: " + err.Error()})
		return
	}

	if err := h.queue.Enqueue(c.Request.Context(), queue.EmailSendingQueue, string(payloadBytes)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to queue email: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, Response{Data: gin.H{"message": "Email queued for sending.", "record_id": record.ID}})
}
