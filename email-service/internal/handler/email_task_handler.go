package handler

import (
	"context"
	"email-service/internal/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

// EmailTaskService defines the interface for email task-related business logic.
type EmailTaskService interface {
	CreateBatchSendTask(
		ctx context.Context,
		taskName string,
		accountSenderID int64,
		recipientIDs []uint,
		templateID *int64,
		subject, body string,
	) (*model.EmailTask, error)
}

// EmailTaskHandler handles HTTP requests for email tasks.
type EmailTaskHandler struct {
	service EmailTaskService
}

// NewEmailTaskHandler creates a new EmailTaskHandler.
func NewEmailTaskHandler(service EmailTaskService) *EmailTaskHandler {
	return &EmailTaskHandler{service: service}
}

// CreateBatchTaskRequest defines the request body for creating a new batch send task.
// The user must provide either a template_id or both subject and body.
type CreateBatchTaskRequest struct {
	TaskName        string `json:"task_name" binding:"required"`
	AccountSenderID int64  `json:"account_sender_id" binding:"required"`
	RecipientIDs    []uint `json:"recipient_ids" binding:"required,min=1"`
	TemplateID      *int64 `json:"template_id"`
	Subject         string `json:"subject"`
	Body            string `json:"body"`
}

// CreateBatchTask handles the API endpoint for creating a batch sending task.
func (h *EmailTaskHandler) CreateBatchTask(c *gin.Context) {
	var req CreateBatchTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.service.CreateBatchSendTask(
		c.Request.Context(),
		req.TaskName,
		req.AccountSenderID,
		req.RecipientIDs,
		req.TemplateID,
		req.Subject,
		req.Body,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Batch task created and has been queued for dispatching.",
		"task_id": task.ID,
	})
}
