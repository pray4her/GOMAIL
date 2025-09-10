package service

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/repository"
	"fmt"
	"time"
)

// TaskSummary represents the aggregated statistical data for an email task.
// @Description Provides a high-level overview of an email task's performance.
type TaskSummary struct {
	ID               int64   `json:"id" example:"1"`
	TaskName         string  `json:"task_name" example:"Q3 Newsletter"`
	Status           string  `json:"status" example:"completed"`
	TotalRecipients  int     `json:"total_recipients" example:"1500"`
	OpenCount        int     `json:"open_count" example:"800"`
	ClickCount       int     `json:"click_count" example:"250"`
	UniqueOpenCount  int     `json:"unique_open_count" example:"600"`
	UniqueClickCount int     `json:"unique_click_count" example:"200"`
	OpenRate         float64 `json:"open_rate" example:"0.53"`
	ClickRate        float64 `json:"click_rate" example:"0.16"`
	UniqueOpenRate   float64 `json:"unique_open_rate" example:"0.4"`
	UniqueClickRate  float64 `json:"unique_click_rate" example:"0.13"`
	CreatedAt        string  `json:"created_at" example:"2023-10-26 10:00:00"`
}

// StatisticsRequest 统计查询请求参数
type StatisticsRequest struct {
	StartDate       *string `form:"start_date" json:"start_date" example:"2023-10-01"`
	EndDate         *string `form:"end_date" json:"end_date" example:"2023-10-31"`
	AccountID       *int64  `form:"account_id" json:"account_id" example:"1"`
	AccountSenderID *int64  `form:"account_sender_id" json:"account_sender_id" example:"1"`
	GroupBy         string  `form:"group_by" json:"group_by" example:"day" enums:"day,week,month,sender"`
}

// StatisticsResponse 统计分析响应结构
type StatisticsResponse struct {
	Summary    *repository.OverallStatistics  `json:"summary"`
	TimeSeries []*repository.DailyStatistics  `json:"time_series,omitempty"`
	BySender   []*repository.SenderStatistics `json:"by_sender,omitempty"`
	Period     *StatisticsPeriod              `json:"period"`
}

// StatisticsPeriod 统计周期信息
type StatisticsPeriod struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Days      int    `json:"days"`
}

// StatisticsService provides methods for querying email statistics.
type StatisticsService struct {
	recordRepo     repository.EmailSendRecordRepository
	taskRepo       repository.EmailTaskRepository
	statisticsRepo repository.SendStatisticsRepository
}

// NewStatisticsService creates a new StatisticsService.
func NewStatisticsService(recordRepo repository.EmailSendRecordRepository, taskRepo repository.EmailTaskRepository, statisticsRepo repository.SendStatisticsRepository) *StatisticsService {
	return &StatisticsService{
		recordRepo:     recordRepo,
		taskRepo:       taskRepo,
		statisticsRepo: statisticsRepo,
	}
}

// GetTaskSummary retrieves and calculates the summary for a specific task.
func (s *StatisticsService) GetTaskSummary(ctx context.Context, taskID int64) (*TaskSummary, error) {
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to find task with id %d: %w", taskID, err)
	}

	totalRecipients := task.TotalRecipients

	summary := &TaskSummary{
		ID:               task.ID,
		TaskName:         task.TaskName,
		Status:           task.Status,
		TotalRecipients:  totalRecipients,
		OpenCount:        task.OpenCount,
		ClickCount:       task.ClickCount,
		UniqueOpenCount:  task.UniqueOpenCount,
		UniqueClickCount: task.UniqueClickCount,
		OpenRate:         task.OpenRate,
		ClickRate:        task.ClickRate,
		UniqueOpenRate:   task.UniqueOpenRate,
		UniqueClickRate:  task.UniqueClickRate,
		CreatedAt:        task.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	return summary, nil
}

// GetTaskRecords returns all send records associated with a specific task ID.
func (s *StatisticsService) GetTaskRecords(ctx context.Context, taskID int64) ([]*model.EmailSendRecord, error) {
	// In a more complex scenario, you might check user permissions here to ensure
	// the user making the request is allowed to see the status of this task.
	return s.recordRepo.FindByTaskID(taskID)
}

// GetMultiDimensionalStatistics 获取多维度统计分析数据
func (s *StatisticsService) GetMultiDimensionalStatistics(ctx context.Context, req *StatisticsRequest) (*StatisticsResponse, error) {
	// 解析和验证日期参数
	filter, err := s.buildStatisticsFilter(req)
	if err != nil {
		return nil, fmt.Errorf("invalid request parameters: %w", err)
	}

	// 构建响应结构
	response := &StatisticsResponse{}

	// 设置周期信息
	response.Period = s.buildPeriodInfo(filter)

	// 获取整体统计数据
	overallStats, err := s.statisticsRepo.GetOverallStatistics(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall statistics: %w", err)
	}
	response.Summary = overallStats

	// 根据groupBy参数决定返回的数据类型
	switch req.GroupBy {
	case "day", "week", "month", "":
		// 获取时间序列数据
		timeSeries, err := s.statisticsRepo.GetDailyStatistics(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get time series statistics: %w", err)
		}
		response.TimeSeries = timeSeries

		// 同时获取发件人统计（作为附加信息）
		senderStats, err := s.statisticsRepo.GetSenderStatistics(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get sender statistics: %w", err)
		}
		response.BySender = senderStats

	case "sender":
		// 只获取按发件人分组的统计
		senderStats, err := s.statisticsRepo.GetSenderStatistics(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get sender statistics: %w", err)
		}
		response.BySender = senderStats
	}

	return response, nil
}

// buildStatisticsFilter 构建统计查询过滤器
func (s *StatisticsService) buildStatisticsFilter(req *StatisticsRequest) (*repository.StatisticsFilter, error) {
	filter := &repository.StatisticsFilter{
		GroupBy: req.GroupBy,
	}

	// 解析开始日期
	if req.StartDate != nil && *req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start_date format, expected YYYY-MM-DD: %w", err)
		}
		filter.StartDate = &startDate
	}

	// 解析结束日期
	if req.EndDate != nil && *req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format, expected YYYY-MM-DD: %w", err)
		}
		filter.EndDate = &endDate
	}

	// 如果没有指定日期范围，默认为最近30天
	if filter.StartDate == nil && filter.EndDate == nil {
		now := time.Now()
		thirtyDaysAgo := now.AddDate(0, 0, -30)
		filter.StartDate = &thirtyDaysAgo
		filter.EndDate = &now
	}

	// 设置其他过滤条件
	if req.AccountID != nil {
		filter.AccountID = req.AccountID
	}

	if req.AccountSenderID != nil {
		filter.AccountSenderID = req.AccountSenderID
	}

	return filter, nil
}

// buildPeriodInfo 构建周期信息
func (s *StatisticsService) buildPeriodInfo(filter *repository.StatisticsFilter) *StatisticsPeriod {
	period := &StatisticsPeriod{}

	if filter.StartDate != nil {
		period.StartDate = filter.StartDate.Format("2006-01-02")
	}
	if filter.EndDate != nil {
		period.EndDate = filter.EndDate.Format("2006-01-02")
	}

	// 计算天数
	if filter.StartDate != nil && filter.EndDate != nil {
		duration := filter.EndDate.Sub(*filter.StartDate)
		period.Days = int(duration.Hours()/24) + 1
	}

	return period
}
