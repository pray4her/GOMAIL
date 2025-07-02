package handler

import (
	"email-service/internal/model"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RecipientService defines the interface for recipient-related business logic.
type RecipientService interface {
	CreateRecipient(email string, firstName, lastName *string, metadata map[string]interface{}) (*model.Recipient, error)
	GetRecipient(id uint) (*model.Recipient, error)
	ListRecipients(page, pageSize int) ([]model.Recipient, int64, error)
	UpdateRecipient(id uint, email *string, firstName, lastName *string, status *string, metadata map[string]interface{}) (*model.Recipient, error)
	DeleteRecipient(id uint) error
}

// RecipientHandler handles HTTP requests for recipients.
type RecipientHandler struct {
	service RecipientService
}

// NewRecipientHandler creates a new RecipientHandler.
func NewRecipientHandler(service RecipientService) *RecipientHandler {
	return &RecipientHandler{service: service}
}

type CreateRecipientRequest struct {
	Email     string                 `json:"email" binding:"required,email"`
	FirstName *string                `json:"first_name"`
	LastName  *string                `json:"last_name"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (h *RecipientHandler) CreateRecipient(c *gin.Context) {
	var req CreateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipient, err := h.service.CreateRecipient(req.Email, req.FirstName, req.LastName, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, recipient)
}

func (h *RecipientHandler) GetRecipient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipient ID"})
		return
	}
	recipient, err := h.service.GetRecipient(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recipient)
}

func (h *RecipientHandler) ListRecipients(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	recipients, total, err := h.service.ListRecipients(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      recipients,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

type UpdateRecipientRequest struct {
	Email     *string                `json:"email" binding:"omitempty,email"`
	FirstName *string                `json:"first_name"`
	LastName  *string                `json:"last_name"`
	Status    *string                `json:"status"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func (h *RecipientHandler) UpdateRecipient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipient ID"})
		return
	}

	var req UpdateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipient, err := h.service.UpdateRecipient(uint(id), req.Email, req.FirstName, req.LastName, req.Status, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, recipient)
}

func (h *RecipientHandler) DeleteRecipient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid recipient ID"})
		return
	}
	if err := h.service.DeleteRecipient(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
