package handler

import (
	"email-service/internal/middleware"
	"email-service/internal/model"
	"email-service/internal/service"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// RecipientService defines the interface for recipient-related business logic.
type RecipientService interface {
	CreateRecipient(email string, firstName, lastName *string, metadata map[string]interface{}) (*model.Recipient, error)
	GetRecipient(id uint) (*model.Recipient, error)
	ListRecipients(page, pageSize int, filters map[string]string) ([]model.Recipient, int64, error)
	ListRecipientsWithSearchAfter(searchAfter []interface{}, pageSize int, filters map[string]string) ([]model.Recipient, []interface{}, int64, error)
	UpdateRecipient(id uint, email *string, firstName, lastName *string, status *string, metadata map[string]interface{}) (*model.Recipient, error)
	DeleteRecipient(id uint) error
}

// RecipientHandler handles HTTP requests for recipients.
type RecipientHandler struct {
	service       RecipientService
	importService service.RecipientImportService
}

// NewRecipientHandler creates a new RecipientHandler.
func NewRecipientHandler(service RecipientService, importService service.RecipientImportService) *RecipientHandler {
	return &RecipientHandler{
		service:       service,
		importService: importService,
	}
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
// @Description  Retrieves a paginated list of all recipients. Can be filtered by name (fuzzy), email (exact), and metadata fields. Supports both traditional pagination (page/pageSize) and search_after pagination for deep pages.
// @Tags         Recipients
// @Produce      json
// @Param        page         query     int    false  "Page number (default: 1) - for traditional pagination"
// @Param        pageSize     query     int    false  "Number of items per page (default: 10) - for traditional pagination"
// @Param        search_after query     string false  "Base64 encoded search_after value for deep pagination"
// @Param        page_size    query     int    false  "Number of items per page (default: 50) - for search_after pagination"
// @Param        name         query     string false  "Filter by recipient's first or last name (fuzzy search)."
// @Param        email        query     string false  "Filter by recipient's email address (exact match)."
// @Param        metadata.    query     string false "Filter by metadata field, e.g., metadata.promo_code=WINTER25"
// @Success      200          {object}  PaginatedResponse{data=[]model.Recipient}
// @Failure      400          {object}  Response
// @Failure      500          {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients [get]
func (h *RecipientHandler) ListRecipients(c *gin.Context) {
	// Parse pagination parameters
	searchAfter, pageSize, filters, useSearchAfter, err := h.parseListParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid search_after parameter: " + err.Error()})
		return
	}

	if useSearchAfter {
		// Use search_after pagination for deep pages
		recipients, nextSearchAfter, total, err := h.service.ListRecipientsWithSearchAfter(searchAfter, pageSize, filters)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
			return
		}

		// Encode next search_after value
		nextSearchAfterEncoded, err := h.encodeSearchAfter(nextSearchAfter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{Error: "Failed to encode search_after: " + err.Error()})
			return
		}

		// Determine if there are more pages
		hasNext := len(recipients) == pageSize && len(nextSearchAfter) > 0

		c.JSON(http.StatusOK, PaginatedResponse{
			Data:       recipients,
			Pagination: NewSearchAfterPagination(pageSize, total, nextSearchAfterEncoded, hasNext),
		})
	} else {
		// Use traditional pagination
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))

		recipients, total, err := h.service.ListRecipients(page, pageSize, filters)
		if err != nil {
			// Check if this is a deep pagination error
			if strings.Contains(err.Error(), "deep pagination detected") {
				c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
			return
		}

		c.JSON(http.StatusOK, PaginatedResponse{
			Data:       recipients,
			Pagination: NewPagination(page, pageSize, total),
		})
	}
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

// BatchUpload handles file upload for batch recipient creation.
// @Summary      Batch Upload Recipients
// @Description  Upload CSV, Excel, or JSON file to create multiple recipients.
// @Tags         Recipients
// @Accept       multipart/form-data
// @Produce      json
// @Param        file       formData  file    true  "Recipients file (CSV, Excel, or JSON)"
// @Param        task_name  formData  string  true  "Name for this import task"
// @Success      202        {object}  Response{data=model.RecipientImportTask}
// @Failure      400        {object}  Response
// @Failure      500        {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients/batch-upload [post]
func (h *RecipientHandler) BatchUpload(c *gin.Context) {
	// Get user ID from JWT token (assuming it's set by auth middleware)
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{Error: "User not authenticated"})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{Error: "Invalid user ID"})
		return
	}

	// Get task name from form
	taskName := c.PostForm("task_name")
	if taskName == "" {
		c.JSON(http.StatusBadRequest, Response{Error: "task_name is required"})
		return
	}

	// Get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "file is required"})
		return
	}

	// Validate file size (50MB limit)
	// const maxFileSize = 50 * 1024 * 1024 // 50MB
	// if fileHeader.Size > maxFileSize {
	// 	c.JSON(http.StatusBadRequest, Response{Error: "File size exceeds 50MB limit"})
	// 	return
	// }

	// Create import task
	task, err := h.importService.CreateImportTask(userIDInt, taskName, fileHeader.Filename, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, Response{Data: task})
}

// GetImportTask retrieves import task status and progress.
// @Summary      Get Import Task
// @Description  Retrieves the status and progress of a batch import task.
// @Tags         Recipients
// @Produce      json
// @Param        id   path      int  true  "Import Task ID"
// @Success      200  {object}  Response{data=model.RecipientImportTask}
// @Failure      400  {object}  Response  "Invalid task ID"
// @Failure      404  {object}  Response  "Import task not found"
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients/import-tasks/{id} [get]
func (h *RecipientHandler) GetImportTask(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid import task ID"})
		return
	}

	task, err := h.importService.GetImportTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Data: task})
}

// ListImportTasks lists user's import tasks.
// @Summary      List Import Tasks
// @Description  Retrieves a paginated list of the current user's import tasks.
// @Tags         Recipients
// @Produce      json
// @Param        page      query     int    false  "Page number (default: 1)"
// @Param        pageSize  query     int    false  "Number of items per page (default: 10)"
// @Success      200       {object}  PaginatedResponse{data=[]model.RecipientImportTask}
// @Failure      500       {object}  Response
// @Security     ApiKeyAuth
// @Router       /api/v1/recipients/import-tasks [get]
func (h *RecipientHandler) ListImportTasks(c *gin.Context) {
	// Get user ID from JWT token
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{Error: "User not authenticated"})
		return
	}

	userIDInt, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusUnauthorized, Response{Error: "Invalid user ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	tasks, total, err := h.importService.ListImportTasks(userIDInt, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Data: tasks,
		Pagination: Pagination{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// DownloadSampleCSV provides a sample CSV file for batch recipient import.
// @Summary      Download Sample CSV
// @Description  Downloads a sample CSV file with headers and example data to demonstrate the required format for batch uploads.
// @Tags         Recipients
// @Produce      text/csv
// @Success      200  {file}  string  "CSV file"
// @Header       200  {string}  Content-Disposition  "attachment; filename=\"sample_recipients.csv\""
// @Router       /api/v1/recipients/batch-upload/sample/csv [get]
func (h *RecipientHandler) DownloadSampleCSV(c *gin.Context) {
	c.Header("Content-Disposition", "attachment; filename=\"sample_recipients.csv\"")
	c.Header("Content-Type", "text/csv")

	// Using a simple string builder for this static content is efficient.
	var sb strings.Builder
	sb.WriteString("email,first_name,last_name,status,metadata\n")
	sb.WriteString("john.doe@example.com,John,Doe,subscribed,\"{}\"\n")
	sb.WriteString("jane.smith@example.com,Jane,Smith,unsubscribed,\"{ \"\"source\"\": \"\"import\"\", \"\"vip\"\": true }\"\n")
	sb.WriteString("test.user@example.com,,,subscribed,\n")

	c.String(http.StatusOK, sb.String())
}

// DownloadSampleJSON provides a sample JSON file for batch recipient import.
// @Summary      Download Sample JSON
// @Description  Downloads a sample JSON file with the required structure and example data for batch uploads.
// @Tags         Recipients
// @Produce      application/json
// @Success      200  {object}  object  "JSON file"
// @Header       200  {string}  Content-Disposition  "attachment; filename=\"sample_recipients.json\""
// @Router       /api/v1/recipients/batch-upload/sample/json [get]
func (h *RecipientHandler) DownloadSampleJSON(c *gin.Context) {
	c.Header("Content-Disposition", "attachment; filename=\"sample_recipients.json\"")
	c.Header("Content-Type", "application/json")

	sampleData := gin.H{
		"recipients": []gin.H{
			{
				"email":      "john.doe@example.com",
				"first_name": "John",
				"last_name":  "Doe",
				"status":     "subscribed",
				"metadata":   gin.H{},
			},
			{
				"email":      "jane.smith@example.com",
				"first_name": "Jane",
				"last_name":  "Smith",
				"status":     "unsubscribed",
				"metadata": gin.H{
					"source": "import",
					"vip":    true,
				},
			},
			{
				"email":  "test.user@example.com",
				"status": "subscribed",
			},
		},
	}

	c.JSON(http.StatusOK, sampleData)
}

// encodeSearchAfter encodes search_after values to Base64 string
func (h *RecipientHandler) encodeSearchAfter(searchAfter []interface{}) (string, error) {
	if len(searchAfter) == 0 {
		return "", nil
	}
	jsonBytes, err := json.Marshal(searchAfter)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(jsonBytes), nil
}

// decodeSearchAfter decodes Base64 string to search_after values
func (h *RecipientHandler) decodeSearchAfter(encoded string) ([]interface{}, error) {
	if encoded == "" {
		return nil, nil
	}
	jsonBytes, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var searchAfter []interface{}
	err = json.Unmarshal(jsonBytes, &searchAfter)
	return searchAfter, err
}

// parseListParams validates and parses pagination parameters
func (h *RecipientHandler) parseListParams(c *gin.Context) (searchAfter []interface{}, pageSize int, filters map[string]string, useSearchAfter bool, err error) {
	queryValues := c.Request.URL.Query()

	// Check if search_after parameter is present
	searchAfterParam := queryValues.Get("search_after")
	useSearchAfter = searchAfterParam != ""

	if useSearchAfter {
		searchAfter, err = h.decodeSearchAfter(searchAfterParam)
		if err != nil {
			return nil, 0, nil, false, err
		}
		pageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "50"))
	} else {
		pageSize, _ = strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	}

	// Extract filter parameters
	filters = make(map[string]string)
	if name := queryValues.Get("name"); name != "" {
		filters["name"] = name
	}
	if email := queryValues.Get("email"); email != "" {
		filters["email"] = email
	}

	for key, values := range queryValues {
		if strings.HasPrefix(key, "metadata.") {
			if len(values) > 0 {
				filters[key] = values[0]
			}
		}
	}

	return searchAfter, pageSize, filters, useSearchAfter, nil
}
