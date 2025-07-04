package repository

import (
	"email-service/internal/model"
	"time"

	"gorm.io/gorm"
)

// StatisticsFilter 统计查询过滤条件
type StatisticsFilter struct {
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	AccountID       *int64     `json:"account_id"`
	AccountSenderID *int64     `json:"account_sender_id"`
	GroupBy         string     `json:"group_by"` // day, week, month, account, sender
}

// DailyStatistics 日级统计数据
type DailyStatistics struct {
	Date             time.Time `json:"date"`
	SentCount        int       `json:"sent_count"`
	OpenCount        int       `json:"open_count"`
	UniqueOpenCount  int       `json:"unique_open_count"`
	ClickCount       int       `json:"click_count"`
	UniqueClickCount int       `json:"unique_click_count"`
	OpenRate         float64   `json:"open_rate"`
	ClickRate        float64   `json:"click_rate"`
	UniqueOpenRate   float64   `json:"unique_open_rate"`
	UniqueClickRate  float64   `json:"unique_click_rate"`
}

// SenderStatistics 发件人统计数据
type SenderStatistics struct {
	AccountSenderID  int64   `json:"account_sender_id"`
	SenderEmail      string  `json:"sender_email"`
	SenderName       string  `json:"sender_name"`
	AccountName      string  `json:"account_name"`
	SentCount        int     `json:"sent_count"`
	OpenCount        int     `json:"open_count"`
	UniqueOpenCount  int     `json:"unique_open_count"`
	ClickCount       int     `json:"click_count"`
	UniqueClickCount int     `json:"unique_click_count"`
	OpenRate         float64 `json:"open_rate"`
	ClickRate        float64 `json:"click_rate"`
	UniqueOpenRate   float64 `json:"unique_open_rate"`
	UniqueClickRate  float64 `json:"unique_click_rate"`
}

// OverallStatistics 整体统计数据
type OverallStatistics struct {
	TotalSent              int     `json:"total_sent"`
	TotalOpened            int     `json:"total_opened"`
	TotalUniqueOpened      int     `json:"total_unique_opened"`
	TotalClicked           int     `json:"total_clicked"`
	TotalUniqueClicked     int     `json:"total_unique_clicked"`
	OverallOpenRate        float64 `json:"overall_open_rate"`
	OverallClickRate       float64 `json:"overall_click_rate"`
	OverallUniqueOpenRate  float64 `json:"overall_unique_open_rate"`
	OverallUniqueClickRate float64 `json:"overall_unique_click_rate"`
}

// SendStatisticsRepository 统计数据仓库接口
type SendStatisticsRepository interface {
	// 获取整体统计数据
	GetOverallStatistics(filter *StatisticsFilter) (*OverallStatistics, error)

	// 获取时间序列统计数据
	GetDailyStatistics(filter *StatisticsFilter) ([]*DailyStatistics, error)

	// 获取发件人维度统计数据
	GetSenderStatistics(filter *StatisticsFilter) ([]*SenderStatistics, error)

	// 获取原始统计记录
	FindByFilter(filter *StatisticsFilter) ([]*model.SendStatistics, error)

	// 创建或更新统计记录
	CreateOrUpdate(stats *model.SendStatistics) error

	// 获取今日发送的邮件总数
	GetTotalSentCountForToday(accountSenderIDs []int64) (map[int64]int, error)
}

type sendStatisticsRepository struct {
	db *gorm.DB
}

// NewSendStatisticsRepository 创建新的统计仓库实例
func NewSendStatisticsRepository(db *gorm.DB) SendStatisticsRepository {
	return &sendStatisticsRepository{db: db}
}

// GetOverallStatistics now queries the aggregated send_statistics table.
func (r *sendStatisticsRepository) GetOverallStatistics(filter *StatisticsFilter) (*OverallStatistics, error) {
	var result OverallStatistics

	query := r.db.Model(&model.SendStatistics{}).
		Select(`
			COALESCE(SUM(sent_count), 0) as total_sent,
			COALESCE(SUM(open_count), 0) as total_opened,
			COALESCE(SUM(unique_open_count), 0) as total_unique_opened,
			COALESCE(SUM(click_count), 0) as total_clicked,
			COALESCE(SUM(unique_click_count), 0) as total_unique_clicked
		`)

	query = r.applyFilter(query, filter)

	if err := query.Scan(&result).Error; err != nil {
		return nil, err
	}

	// Calculate rates
	if result.TotalSent > 0 {
		result.OverallOpenRate = float64(result.TotalOpened) / float64(result.TotalSent)
		result.OverallClickRate = float64(result.TotalClicked) / float64(result.TotalSent)
		result.OverallUniqueOpenRate = float64(result.TotalUniqueOpened) / float64(result.TotalSent)
		result.OverallUniqueClickRate = float64(result.TotalUniqueClicked) / float64(result.TotalSent)
	}

	return &result, nil
}

// GetDailyStatistics now queries the aggregated send_statistics table.
func (r *sendStatisticsRepository) GetDailyStatistics(filter *StatisticsFilter) ([]*DailyStatistics, error) {
	var results []*DailyStatistics

	query := r.db.Model(&model.SendStatistics{}).
		Select(`
			stat_date as date,
			COALESCE(SUM(sent_count), 0) as sent_count,
			COALESCE(SUM(open_count), 0) as open_count,
			COALESCE(SUM(unique_open_count), 0) as unique_open_count,
			COALESCE(SUM(click_count), 0) as click_count,
			COALESCE(SUM(unique_click_count), 0) as unique_click_count
		`).
		Group("stat_date").
		Order("stat_date ASC")

	query = r.applyFilter(query, filter)

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Calculate rates for each daily entry
	for _, daily := range results {
		if daily.SentCount > 0 {
			daily.OpenRate = float64(daily.OpenCount) / float64(daily.SentCount)
			daily.ClickRate = float64(daily.ClickCount) / float64(daily.SentCount)
			daily.UniqueOpenRate = float64(daily.UniqueOpenCount) / float64(daily.SentCount)
			daily.UniqueClickRate = float64(daily.UniqueClickCount) / float64(daily.SentCount)
		}
	}

	return results, nil
}

// GetSenderStatistics now queries the aggregated send_statistics table.
func (r *sendStatisticsRepository) GetSenderStatistics(filter *StatisticsFilter) ([]*SenderStatistics, error) {
	var results []*SenderStatistics

	query := r.db.Model(&model.SendStatistics{}).
		Select(`
			send_statistics.account_sender_id,
			account_senders.email_address as sender_email,
			senders.name as sender_name,
			accounts.name as account_name,
			COALESCE(SUM(send_statistics.sent_count), 0) as sent_count,
			COALESCE(SUM(send_statistics.open_count), 0) as open_count,
			COALESCE(SUM(send_statistics.unique_open_count), 0) as unique_open_count,
			COALESCE(SUM(send_statistics.click_count), 0) as click_count,
			COALESCE(SUM(send_statistics.unique_click_count), 0) as unique_click_count
		`).
		Joins("LEFT JOIN account_senders ON send_statistics.account_sender_id = account_senders.id").
		Joins("LEFT JOIN senders ON account_senders.sender_id = senders.id").
		Joins("LEFT JOIN accounts ON send_statistics.account_id = accounts.id").
		Group("send_statistics.account_sender_id, account_senders.email_address, senders.name, accounts.name").
		Order("sent_count DESC")

	query = r.applyFilter(query, filter)

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Calculate rates for each sender entry
	for _, sender := range results {
		if sender.SentCount > 0 {
			sender.OpenRate = float64(sender.OpenCount) / float64(sender.SentCount)
			sender.ClickRate = float64(sender.ClickCount) / float64(sender.SentCount)
			sender.UniqueOpenRate = float64(sender.UniqueOpenCount) / float64(sender.SentCount)
			sender.UniqueClickRate = float64(sender.UniqueClickCount) / float64(sender.SentCount)
		}
	}

	return results, nil
}

// FindByFilter 根据过滤条件查找统计记录
func (r *sendStatisticsRepository) FindByFilter(filter *StatisticsFilter) ([]*model.SendStatistics, error) {
	var records []*model.SendStatistics

	query := r.db.Preload("Account").Preload("AccountSender").Preload("AccountSender.Sender")
	query = r.applyFilter(query, filter)

	err := query.Order("stat_date DESC").Find(&records).Error
	return records, err
}

// GetTotalSentCountForToday returns a map of account sender IDs to their total sent count for the current day.
func (r *sendStatisticsRepository) GetTotalSentCountForToday(accountSenderIDs []int64) (map[int64]int, error) {
	results := make(map[int64]int)
	if len(accountSenderIDs) == 0 {
		return results, nil
	}

	type dailySent struct {
		AccountSenderID int64
		SentCount       int
	}

	var dailyCounts []dailySent

	// Get the start of the current day in UTC.
	// IMPORTANT: Ensure your database stores stat_date consistently, preferably as a DATE type without time.
	today := time.Now().UTC()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	err := r.db.Model(&model.SendStatistics{}).
		Select("account_sender_id, sent_count").
		Where("account_sender_id IN ?", accountSenderIDs).
		Where("stat_date = ?", startOfDay).
		Scan(&dailyCounts).Error

	if err != nil {
		return nil, err
	}

	for _, count := range dailyCounts {
		results[count.AccountSenderID] = count.SentCount
	}

	return results, nil
}

// CreateOrUpdate 创建或更新统计记录
func (r *sendStatisticsRepository) CreateOrUpdate(stats *model.SendStatistics) error {
	// Find existing record for the same day and sender
	var existing model.SendStatistics
	err := r.db.Where("stat_date = ? AND account_sender_id = ?", stats.StatDate, stats.AccountSenderID).First(&existing).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return err // Real error
	}

	if err == gorm.ErrRecordNotFound {
		// Create new record
		return r.db.Create(stats).Error
	}

	// Update existing record by adding new counts
	return r.db.Model(&existing).Updates(model.SendStatistics{
		SentCount:        existing.SentCount + stats.SentCount,
		OpenCount:        existing.OpenCount + stats.OpenCount,
		UniqueOpenCount:  existing.UniqueOpenCount + stats.UniqueOpenCount,
		ClickCount:       existing.ClickCount + stats.ClickCount,
		UniqueClickCount: existing.UniqueClickCount + stats.UniqueClickCount,
	}).Error
}

// applyFilter applies date and ID filters to a query on the send_statistics table.
func (r *sendStatisticsRepository) applyFilter(query *gorm.DB, filter *StatisticsFilter) *gorm.DB {
	if filter.StartDate != nil {
		query = query.Where("stat_date >= ?", filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("stat_date <= ?", filter.EndDate)
	}
	if filter.AccountID != nil {
		query = query.Where("account_id = ?", filter.AccountID)
	}
	if filter.AccountSenderID != nil {
		query = query.Where("account_sender_id = ?", filter.AccountSenderID)
	}
	return query
}
