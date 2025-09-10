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

// TemplateHandler handles HTTP requests for email templates.
type TemplateHandler struct {
	service service.TemplateService
}

func NewTemplateHandler(service service.TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

// ListTemplatesResponse defines the successful response structure for listing templates.
type ListTemplatesResponse struct {
	Templates  []model.EmailTemplate `json:"templates"`
	Pagination Pagination            `json:"pagination"`
}

// CreateTemplateRequest defines the request body for creating a template.
// @Description Request body for creating a new email template.
type CreateTemplateRequest struct {
	Name    string `json:"name" binding:"required" example:"Welcome Email"`
	Subject string `json:"subject" binding:"required" example:"Welcome to Our Service!"`
	Body    string `json:"body" binding:"required" example:"<h1>Hello {{.FirstName}}!</h1><p>Thank you for joining us.</p>"`
}

// UpdateTemplateRequest defines the request body for updating a template.
// @Description Request body for updating an existing email template.
type UpdateTemplateRequest struct {
	Name    string `json:"name,omitempty" example:"Welcome Email V2"`
	Subject string `json:"subject,omitempty" example:"A Warm Welcome to Our Service!"`
	Body    string `json:"body,omitempty" example:"<h2>Hello {{.FirstName}}!</h2><p>We are thrilled to have you.</p>"`
}

// CreateTemplate handles the creation of a new email template.
// @Summary      Create Template
// @Description  Creates a new email template.
// @Tags         Templates
// @Accept       json
// @Produce      json
// @Param        template  body      CreateTemplateRequest  true  "Template Details"
// @Success      201       {object}  Response{data=model.EmailTemplate}
// @Failure      400       {object}  Response
// @Failure      500       {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/templates [post]
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	var template model.EmailTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	if err := h.service.CreateTemplate(&template); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to create template"})
		return
	}

	c.JSON(http.StatusCreated, Response{Data: template})
}

// GetTemplates retrieves email templates with optional pagination.
// @Summary      Get Templates
// @Description  Retrieves a paginated list of email templates. If page and pageSize are not provided, returns all templates.
// @Tags         Templates
// @Produce      json
// @Param        page      query     int  false  "Page number for pagination"  default(1)
// @Param        pageSize  query     int  false  "Number of templates per page"  default(10)
// @Success      200       {object}  Response{data=ListTemplatesResponse}
// @Failure      400       {object}  Response  "Invalid page or pageSize parameters"
// @Failure      500       {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/templates [get]
func (h *TemplateHandler) GetTemplates(c *gin.Context) {
	// Check if pagination parameters are provided
	pageStr := c.Query("page")
	pageSizeStr := c.Query("pageSize")

	// If no pagination parameters, return all templates (backward compatibility)
	if pageStr == "" && pageSizeStr == "" {
		templates, err := h.service.GetTemplates()
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: "Failed to retrieve templates"})
			return
		}
		c.JSON(http.StatusOK, Response{Data: templates})
		return
	}

	// Parse pagination parameters
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

	// Get paginated templates
	templates, total, err := h.service.ListTemplates(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to retrieve templates"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Data: ListTemplatesResponse{
			Templates:  templates,
			Pagination: NewPagination(page, pageSize, total),
		},
	})
}

// GetTemplate retrieves a single email template by its ID.
// @Summary      Get Template by ID
// @Description  Retrieves a specific email template by its ID.
// @Tags         Templates
// @Produce      json
// @Param        id   path      int  true  "Template ID"
// @Success      200  {object}  Response{data=model.EmailTemplate}
// @Failure      400  {object}  Response  "Invalid template ID"
// @Failure      404  {object}  Response  "Template not found"
// @Failure      500  {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/templates/{id} [get]
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid template ID"})
		return
	}
	template, err := h.service.GetTemplateByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{Error: "Template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to retrieve template"})
		return
	}
	c.JSON(http.StatusOK, Response{Data: template})
}

// UpdateTemplate handles the update of an existing email template.
// @Summary      Update Template
// @Description  Updates an existing email template.
// @Tags         Templates
// @Accept       json
// @Produce      json
// @Param        id        path      int                    true  "Template ID"
// @Param        template  body      UpdateTemplateRequest  true  "Updated Template Details"
// @Success      200       {object}  Response{data=model.EmailTemplate}
// @Failure      400       {object}  Response  "Invalid template ID or request body"
// @Failure      500       {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/templates/{id} [put]
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid template ID"})
		return
	}
	var template model.EmailTemplate
	if err := c.ShouldBindJSON(&template); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}
	template.ID = id

	if err := h.service.UpdateTemplate(&template); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to update template"})
		return
	}
	c.JSON(http.StatusOK, Response{Data: template})
}

// DeleteTemplate handles the deletion of an email template.
// @Summary      Delete Template
// @Description  Deletes an email template by its ID.
// @Tags         Templates
// @Produce      json
// @Param        id   path      int  true  "Template ID"
// @Success      200  {object}  Response{data=map[string]string} "message: Template deleted successfully"
// @Failure      400  {object}  Response "Invalid template ID"
// @Failure      500  {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/templates/{id} [delete]
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid template ID"})
		return
	}

	if err := h.service.DeleteTemplate(id); err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to delete template"})
		return
	}
	c.JSON(http.StatusOK, Response{Data: gin.H{"message": "Template deleted successfully"}})
}

// PreviewTemplate handles rendering a preview of a template with sample data.
// @Summary      Preview Template
// @Description  Renders a preview of a template with sample data.
// @Tags         Templates
// @Accept       json
// @Produce      json
// @Param        id   path      int                  true  "Template ID"
// @Param        data body      service.PreviewData  true  "Sample data for preview"
// @Success      200  {object}  Response{data=service.RenderedTemplate}
// @Failure      400  {object}  Response  "Invalid template ID or request body"
// @Failure      404  {object}  Response  "Template not found"
// @Failure      500  {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/templates/{id}/preview [post]
func (h *TemplateHandler) PreviewTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid template ID"})
		return
	}

	var previewData service.PreviewData
	if err := c.ShouldBindJSON(&previewData); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid preview data: " + err.Error()})
		return
	}

	template, err := h.service.GetTemplateByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{Error: "Template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to retrieve template"})
		return
	}

	rendered, err := h.service.RenderPreview(template, previewData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to render template preview: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Data: rendered})
}
