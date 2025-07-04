package service

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/repository"
	"email-service/pkg/aliyun"
	"fmt"
	"log"
	"strconv"
	"time"

	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
)

// TrackingService is responsible for syncing email tracking data from Aliyun.
type TrackingService struct {
	taskRepo       repository.EmailTaskRepository
	recordRepo     repository.EmailSendRecordRepository
	senderRepo     repository.SenderRepository
	statsRepo      repository.SendStatisticsRepository
	aliyunEndpoint string
}

// NewTrackingService creates a new TrackingService.
func NewTrackingService(
	taskRepo repository.EmailTaskRepository,
	recordRepo repository.EmailSendRecordRepository,
	senderRepo repository.SenderRepository,
	statsRepo repository.SendStatisticsRepository,
	aliyunEndpoint string,
) *TrackingService {
	return &TrackingService{
		taskRepo:       taskRepo,
		recordRepo:     recordRepo,
		senderRepo:     senderRepo,
		statsRepo:      statsRepo,
		aliyunEndpoint: aliyunEndpoint,
	}
}

// Start periodically syncs tracking data for ongoing tasks.
func (s *TrackingService) Start(ctx context.Context, interval time.Duration) {
	log.Printf("Starting Task Tracking service with sync interval: %s", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Task Tracking service shutting down.")
			return
		case <-ticker.C:
			log.Println("Running task tracking data sync...")
			if err := s.SyncTaskTrackingData(ctx); err != nil {
				log.Printf("Error during task tracking data sync: %v", err)
			}
		}
	}
}

// SyncTaskTrackingData fetches ongoing tasks and updates their aggregate stats from Aliyun.
func (s *TrackingService) SyncTaskTrackingData(ctx context.Context) error {
	// Find tasks that are in progress and have a tag.
	// The specific statuses might need refinement based on the task lifecycle.
	tasks, err := s.taskRepo.FindTrackableTasks([]string{"pending", "dispatching", "sending"})
	if err != nil {
		return fmt.Errorf("failed to find trackable tasks: %w", err)
	}

	if len(tasks) == 0 {
		log.Println("No trackable tasks found.")
		return nil
	}

	log.Printf("Found %d tasks to track.", len(tasks))

	for _, task := range tasks {
		if task.AliyunTagName == nil || *task.AliyunTagName == "" {
			continue // Should not happen based on query, but as a safeguard.
		}

		// Since a task can now use many senders, we must find a representative sender to get API credentials.
		// We'll fetch the task's records and use the sender from the first one.
		// This assumes the Aliyun tag for the task was created using credentials from a participating sender.
		records, err := s.recordRepo.FindByTaskID(task.ID)
		if err != nil {
			log.Printf("Error fetching records for task %d to find sender: %v. Skipping.", task.ID, err)
			continue
		}
		if len(records) == 0 {
			log.Printf("No records for task %d, cannot determine tracking sender. Skipping.", task.ID)
			continue
		}

		// We need the sender details to initialize the Aliyun client.
		accountSender, err := s.senderRepo.FindAccountSenderDetails(records[0].AccountSenderID)
		if err != nil {
			log.Printf("Error fetching sender details for task %d: %v. Skipping.", task.ID, err)
			continue
		}

		s.updateTaskTracking(accountSender, task)
	}

	return nil
}

func (s *TrackingService) updateTaskTracking(accountSender *model.AccountSender, task *model.EmailTask) {
	client, err := aliyun.NewClient(
		s.aliyunEndpoint,
		accountSender.Account.AccessKeyID,
		accountSender.Account.AccessKeySecret,
	)
	if err != nil {
		log.Printf("Error creating Aliyun client for task %d: %v", task.ID, err)
		return
	}
	aliyunSender := aliyun.NewEmailSender(client)

	startTime := task.CreatedAt.Format("2006-01-02")
	endTime := time.Now().Format("2006-01-02")

	responseBody, err := aliyunSender.GetTrackList(accountSender.EmailAddress, *task.AliyunTagName, startTime, endTime)
	if err != nil {
		log.Printf("Error getting track list for task %d (tag: %s): %v", task.ID, *task.AliyunTagName, err)
		return
	}

	// The GetTrackList now returns the full body. We need to check for nil and the data slice.
	if responseBody == nil || responseBody.Data == nil || responseBody.Data.Stat == nil || len(responseBody.Data.Stat) == 0 {
		return // No data yet.
	}

	stat := responseBody.Data.Stat[0] // Assuming one aggregated stat object per tag.
	s.processAndStoreStatistics(stat, task, accountSender)
}

func (s *TrackingService) processAndStoreStatistics(stat *dm.GetTrackListResponseBodyDataStat, task *model.EmailTask, accountSender *model.AccountSender) {
	// Parse new counts from Aliyun response
	newOpenCount, _ := strconv.Atoi(*stat.RcptOpenCount)
	newClickCount, _ := strconv.Atoi(*stat.RcptClickCount)
	newUniqueOpenCount, _ := strconv.Atoi(*stat.RcptUniqueOpenCount)
	newUniqueClickCount, _ := strconv.Atoi(*stat.RcptUniqueClickCount)
	// newSentCount, _ := strconv.Atoi(*stat.TotalNumber) // This is a cumulative total, sent count is handled on dispatch.

	// Calculate the increment since the last sync for this task
	openIncrement := newOpenCount - task.OpenCount
	clickIncrement := newClickCount - task.ClickCount
	uniqueOpenIncrement := newUniqueOpenCount - task.UniqueOpenCount
	uniqueClickIncrement := newUniqueClickCount - task.UniqueClickCount

	// Sent count is associated with the task creation, not incremental.
	// We handle sent count when the task is dispatched, not here.
	// However, if we need to sync it from Aliyun, a similar incremental logic would apply.

	hasUpdate := openIncrement > 0 || clickIncrement > 0 || uniqueOpenIncrement > 0 || uniqueClickIncrement > 0

	if hasUpdate {
		// 1. Update the aggregate statistics table for today
		today := time.Now().Truncate(24 * time.Hour)
		dailyStat := &model.SendStatistics{
			StatDate:        today,
			AccountID:       accountSender.AccountID,
			AccountSenderID: accountSender.ID,
			// We only record increments. Sent count is handled elsewhere.
			SentCount:        0,
			OpenCount:        openIncrement,
			UniqueOpenCount:  uniqueOpenIncrement,
			ClickCount:       clickIncrement,
			UniqueClickCount: uniqueClickIncrement,
		}

		if err := s.statsRepo.CreateOrUpdate(dailyStat); err != nil {
			log.Printf("Error updating daily aggregate statistics for task %d: %v", task.ID, err)
		} else {
			log.Printf("Successfully updated daily aggregate statistics: Opens=%d, Clicks=%d", openIncrement, clickIncrement)
		}

		// 2. Update the task's own counters to the new total
		task.OpenCount = newOpenCount
		task.ClickCount = newClickCount
		task.UniqueOpenCount = newUniqueOpenCount
		task.UniqueClickCount = newUniqueClickCount
		// Also update rates on the task model
		task.OpenRate, _ = parseFloatRate(*stat.RcptOpenRate)
		task.ClickRate, _ = parseFloatRate(*stat.RcptClickRate)
		task.UniqueOpenRate, _ = parseFloatRate(*stat.RcptUniqueOpenRate)
		task.UniqueClickRate, _ = parseFloatRate(*stat.RcptUniqueClickRate)

		if err := s.taskRepo.Update(task); err != nil {
			log.Printf("Error updating tracking info for task %d: %v", task.ID, err)
		} else {
			log.Printf("Updated tracking for task %d: TotalOpens=%d, TotalClicks=%d", task.ID, task.OpenCount, task.ClickCount)
		}
	}
}

// parseFloatRate safely parses a string rate from Aliyun into a float64.
// Aliyun provides rates as strings, e.g., "0" or "85.34".
func parseFloatRate(rateStr string) (float64, error) {
	if rateStr == "" {
		return 0.0, nil
	}
	return strconv.ParseFloat(rateStr, 64)
}
