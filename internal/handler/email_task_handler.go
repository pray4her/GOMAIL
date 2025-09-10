package handler

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/service"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// EmailTaskService defines the interface for email task-related business logic.
// This should be in sync with the actual service.EmailTaskService interface
type EmailTaskCreator interface {
	CreateEmailTask(
		ctx context.Context,
		userID int64,
		taskName string,
		recipientGroupID int64,
		templateID *int64,
		subject, body string,
		scheduledAt *time.Time,
	) (*model.EmailTask, error)
}

// EmailTaskHandler handles HTTP requests for email tasks.
type EmailTaskHandler struct {
	service service.EmailTaskServiceInterface
}

// NewEmailTaskHandler creates a new EmailTaskHandler.
func NewEmailTaskHandler(service service.EmailTaskServiceInterface) *EmailTaskHandler {
	return &EmailTaskHandler{service: service}
}

// ListTasksResponse defines the successful response structure for listing tasks.
type ListTasksResponse struct {
	Tasks      []model.EmailTask `json:"tasks"`
	Pagination Pagination        `json:"pagination"`
}

// ListTasks handles the API endpoint for listing all email tasks with pagination.
// @Summary      List Email Tasks
// @Description  Retrieves a paginated list of all email tasks.
// @Tags         Tasks
// @Produce      json
// @Param        page      query     int  false  "Page number for pagination"  default(1)
// @Param        pageSize  query     int  false  "Number of tasks per page"    default(10)
// @Success      200       {object}  Response{data=ListTasksResponse}
// @Failure      400       {object}  Response  "Invalid page or pageSize parameters"
// @Failure      500       {object}  Response  "Internal server error"
// @Security     ApiKeyAuth
// @Router       /api/v1/tasks [get]
func (h *EmailTaskHandler) ListTasks(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid page number"})
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid page size"})
		return
	}

	tasks, total, err := h.service.ListTasks(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to retrieve tasks"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Data: ListTasksResponse{
			Tasks:      tasks,
			Pagination: NewPagination(page, pageSize, total),
		},
	})
}

// CreateEmailTaskRequest defines the request body for creating a new batch send task.
type CreateEmailTaskRequest struct {
	TaskName         string     `json:"task_name" binding:"required" example:"Q4 Promotion"`
	RecipientGroupID int64      `json:"recipient_group_id" binding:"required,min=1" example:"5"`
	TemplateID       *int64     `json:"template_id" example:"10"`
	Subject          string     `json:"subject" example:"Special Offer Inside!"`
	Body             string     `json:"body" example:"<p>Dear valued customer, here is a special offer just for you!</p>"`
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty" format:"date-time" example:"2025-01-01T12:00:00Z"`
	SendLimit        *int       `json:"send_limit,omitempty" example:"10000"`
	SendOffset       *int       `json:"send_offset,omitempty" example:"0"`
}

// CreateEmailTask handles the API endpoint for creating a batch sending task.
// @Summary      Create Batch Email Task
// @Description  Creates and queues a new task to send emails to a list of recipients. Either a template_id or a subject/body pair must be provided. This endpoint is asynchronous and will return a task ID upon successful queuing.
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        task  body      CreateEmailTaskRequest  true  "Batch Task Details"
// @Success      201   {object}  Response{data=map[string]interface{}} "message: Email task created..., task_id: 123"
// @Failure      400   {object}  Response  "Invalid request body"
// @Failure      401   {object}  Response  "Unauthorized - User not authenticated"
// @Failure      403   {object}  Response  "Forbidden - User does not have permission to use the specified sender"
// @Failure      500   {object}  Response  "Internal server error"
// @Security     ApiKeyAuth
// @Router       /api/v1/tasks [post]
func (h *EmailTaskHandler) CreateEmailTask(c *gin.Context) {
	var req CreateEmailTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{Error: err.Error()})
		return
	}

	task, err := h.service.CreateEmailTask(
		c.Request.Context(),
		userID,
		req.TaskName,
		req.RecipientGroupID,
		req.TemplateID,
		req.Subject,
		req.Body,
		req.ScheduledAt,
		req.SendLimit,
		req.SendOffset,
	)
	if err != nil {
		if errors.Is(err, service.ErrPermissionDenied) {
			c.JSON(http.StatusForbidden, Response{Error: err.Error()})
			return
		}
		if errors.Is(err, service.ErrValidation) {
			c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
			return
		}
		// Using a generic error message to avoid leaking too much info
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to create email task."})
		return
	}

	c.JSON(http.StatusCreated, Response{Data: gin.H{
		"message": "Email task created and has been queued for dispatching.",
		"task_id": task.ID,
	}})
}

// getUserIDFromContext extracts the user ID from the gin context.
func getUserIDFromContext(c *gin.Context) (int64, error) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		return 0, errors.New("user ID not found in context, middleware might be missing")
	}

	userID, ok := userIDVal.(int64)
	if !ok {
		return 0, errors.New("user ID in context is not of expected type")
	}

	return userID, nil
}
