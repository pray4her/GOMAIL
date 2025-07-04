package handler

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// StatisticsService defines the interface for the statistics service.
type StatisticsService interface {
	GetTaskRecords(ctx context.Context, taskID int64) ([]*model.EmailSendRecord, error)
	GetTaskSummary(ctx context.Context, taskID int64) (*service.TaskSummary, error)
	GetMultiDimensionalStatistics(ctx context.Context, req *service.StatisticsRequest) (*service.StatisticsResponse, error)
}

// StatisticsHandler handles HTTP requests related to statistics.
type StatisticsHandler struct {
	service StatisticsService
}

// NewStatisticsHandler creates a new StatisticsHandler.
func NewStatisticsHandler(service StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{service: service}
}

// GetTaskRecords retrieves all email records for a given task ID.
// @Summary      Get Task Records
// @Description  Get a detailed list of all email send records for a specific task.
// @Tags         Statistics
// @Produce      json
// @Param        id   path      int  true  "Task ID"
// @Success      200  {object}  Response{data=[]model.EmailSendRecord}
// @Failure      400  {object}  Response  "Invalid Task ID"
// @Failure      404  {object}  Response  "Task not found or no records for this task"
// @Failure      500  {object}  Response  "Internal Server Error"
// @Security     ApiKeyAuth
// @Router       /api/v1/tasks/{id}/records [get]
func (h *StatisticsHandler) GetTaskRecords(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid task ID format"})
		return
	}

	records, err := h.service.GetTaskRecords(c.Request.Context(), id)
	if err != nil {
		// This could be a db error or other internal issue.
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	if len(records) == 0 {
		// This isn't an error, but we should inform the client.
		c.JSON(http.StatusOK, Response{Data: records, Error: "No records found for this task ID."})
		return
	}

	c.JSON(http.StatusOK, Response{Data: records})
}

// GetTaskSummary retrieves the summary for a specific task.
// @Summary      Get Task Summary
// @Description  Get a summary of sending statistics for a specific task.
// @Tags         Statistics
// @Produce      json
// @Param        id   path      int                  true  "Task ID"
// @Success      200  {object}  Response{data=service.TaskSummary}  "The data field will contain the task summary"
// @Failure      400  {object}  Response        "Invalid Task ID"
// @Failure      500  {object}  Response        "Internal Server Error"
// @Security     ApiKeyAuth
// @Router       /api/v1/tasks/{id} [get]
func (h *StatisticsHandler) GetTaskSummary(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid task ID format"})
		return
	}

	summary, err := h.service.GetTaskSummary(c.Request.Context(), id)
	if err != nil {
		// Consider specific error types, e.g., not found
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Data: summary})
}

// GetStatistics 获取多维度统计分析数据
// @Summary      Get Multi-Dimensional Statistics
// @Description  Get comprehensive email sending statistics with various filters and grouping options.
// @Description  Provides overall summary, time series data, and sender performance analysis.
// @Tags         Statistics
// @Produce      json
// @Param        start_date        query     string  false  "Start date (YYYY-MM-DD format)"  example(2023-10-01)
// @Param        end_date          query     string  false  "End date (YYYY-MM-DD format)"    example(2023-10-31)
// @Param        account_id        query     int     false  "Filter by specific account ID"   example(1)
// @Param        account_sender_id query     int     false  "Filter by specific sender ID"    example(1)
// @Param        group_by          query     string  false  "Group results by: day, week, month, sender"  Enums(day, week, month, sender)  example(day)
// @Success      200  {object}  Response{data=service.StatisticsResponse}  "The data field contains the comprehensive statistics"
// @Failure      400  {object}  Response               "Invalid request parameters"
// @Failure      500  {object}  Response               "Internal Server Error"
// @Security     ApiKeyAuth
// @Router       /api/v1/statistics [get]
func (h *StatisticsHandler) GetStatistics(c *gin.Context) {
	// 绑定查询参数
	var req service.StatisticsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Error: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// 调用服务层获取统计数据
	statistics, err := h.service.GetMultiDimensionalStatistics(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Error: "Failed to retrieve statistics: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, Response{
		Data: statistics,
	})
}
