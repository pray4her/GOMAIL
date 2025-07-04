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
	Email     string                 `json:"email" binding:"required,email" example:"john.doe@example.com"`
	FirstName *string                `json:"first_name" example:"John"`
	LastName  *string                `json:"last_name" example:"Doe"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// @Summary      Create Recipient
// @Description  Adds a new recipient to the system.
// @Tags         Recipients
// @Accept       json
// @Produce      json
// @Param        recipient  body      CreateRecipientRequest  true  "Recipient Information"
// @Success      201        {object}  Response{data=model.Recipient}
// @Failure      400        {object}  Response
// @Failure      500        {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients [post]
func (h *RecipientHandler) CreateRecipient(c *gin.Context) {
	var req CreateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	recipient, err := h.service.CreateRecipient(req.Email, req.FirstName, req.LastName, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, Response{Data: recipient})
}

// GetRecipient retrieves a single recipient by ID.
// @Summary      Get Recipient by ID
// @Description  Retrieves details for a specific recipient.
// @Tags         Recipients
// @Produce      json
// @Param        id   path      int  true  "Recipient ID"
// @Success      200  {object}  Response{data=model.Recipient}
// @Failure      400  {object}  Response  "Invalid recipient ID"
// @Failure      404  {object}  Response  "Recipient not found"
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients/{id} [get]
func (h *RecipientHandler) GetRecipient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid recipient ID"})
		return
	}
	recipient, err := h.service.GetRecipient(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Data: recipient})
}

// ListRecipients retrieves a paginated list of recipients.
// @Summary      List Recipients
// @Description  Retrieves a paginated list of all recipients.
// @Tags         Recipients
// @Produce      json
// @Param        page      query     int  false  "Page number (default: 1)"
// @Param        pageSize  query     int  false  "Number of items per page (default: 10)"
// @Success      200       {object}  PaginatedResponse{data=[]model.Recipient}
// @Failure      500       {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients [get]
func (h *RecipientHandler) ListRecipients(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	recipients, total, err := h.service.ListRecipients(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, PaginatedResponse{
		Data: recipients,
		Pagination: Pagination{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

type UpdateRecipientRequest struct {
	Email     *string                `json:"email" binding:"omitempty,email" example:"john.doe.updated@example.com"`
	FirstName *string                `json:"first_name" example:"Johnathan"`
	LastName  *string                `json:"last_name" example:"Doe"`
	Status    *string                `json:"status" example:"unsubscribed"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// UpdateRecipient updates an existing recipient.
// @Summary      Update Recipient
// @Description  Updates the details of an existing recipient.
// @Tags         Recipients
// @Accept       json
// @Produce      json
// @Param        id         path      int                     true  "Recipient ID"
// @Param        recipient  body      UpdateRecipientRequest  true  "Recipient fields to update"
// @Success      200        {object}  Response{data=model.Recipient}
// @Failure      400        {object}  Response  "Invalid recipient ID or request body"
// @Failure      500        {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients/{id} [put]
func (h *RecipientHandler) UpdateRecipient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid recipient ID"})
		return
	}

	var req UpdateRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	recipient, err := h.service.UpdateRecipient(uint(id), req.Email, req.FirstName, req.LastName, req.Status, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, Response{Data: recipient})
}

// DeleteRecipient deletes a recipient by ID.
// @Summary      Delete Recipient
// @Description  Deletes a recipient from the system.
// @Tags         Recipients
// @Produce      json
// @Param        id   path      int  true  "Recipient ID"
// @Success      204  {object}  nil "No Content"
// @Failure      400  {object}  Response  "Invalid recipient ID"
// @Failure      500  {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients/{id} [delete]
func (h *RecipientHandler) DeleteRecipient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid recipient ID"})
		return
	}
	if err := h.service.DeleteRecipient(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
