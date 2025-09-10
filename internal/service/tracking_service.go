package service

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/repository"
	"email-service/pkg/aliyun"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
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
			log.Println("Running sent count statistics sync...")
			if err := s.SyncSentCountStatistics(ctx); err != nil {
				log.Printf("Error during sent count statistics sync: %v", err)
			}
		}
	}
}

// SyncTaskTrackingData fetches ongoing tasks and updates their aggregate stats from Aliyun.
func (s *TrackingService) SyncTaskTrackingData(ctx context.Context) error {
	// 1. Find all trackable tasks
	tasks, err := s.taskRepo.FindTrackableTasks([]string{"pending", "dispatching", "sending", "completed"})
	if err != nil {
		return fmt.Errorf("failed to find trackable tasks: %w", err)
	}

	if len(tasks) == 0 {
		log.Println("No trackable tasks found.")
		return nil
	}
	log.Printf("Found %d tasks to track.", len(tasks))

	taskIDs := make([]int64, len(tasks))
	for i, task := range tasks {
		taskIDs[i] = task.ID
	}

	// 2. Batch fetch all relevant records for these tasks
	allRecords, err := s.recordRepo.FindByTaskIDs(taskIDs)
	if err != nil {
		return fmt.Errorf("failed to batch fetch records for tasks: %w", err)
	}

	// 3. Group records by task ID and collect unique sender IDs
	recordsByTaskID := make(map[int64][]*model.EmailSendRecord)
	senderIDSet := make(map[int64]struct{})
	for _, record := range allRecords {
		if record.TaskID != nil {
			recordsByTaskID[*record.TaskID] = append(recordsByTaskID[*record.TaskID], record)
			senderIDSet[record.AccountSenderID] = struct{}{}
		}
	}

	senderIDs := make([]int64, 0, len(senderIDSet))
	for id := range senderIDSet {
		senderIDs = append(senderIDs, id)
	}

	// 4. Batch fetch all unique senders
	allSenders, err := s.senderRepo.FindAccountSenderDetailsByIDs(senderIDs)
	if err != nil {
		return fmt.Errorf("failed to batch fetch sender details: %w", err)
	}
	sendersByID := make(map[int64]*model.AccountSender)
	for _, sender := range allSenders {
		// Note: This creates a new pointer for each sender to avoid issues with loop variable scope.
		s := sender
		sendersByID[sender.ID] = &s
	}

	// 5. Process each task with pre-fetched data
	for _, task := range tasks {
		if task.AliyunTagName == nil || *task.AliyunTagName == "" {
			continue // Should not happen based on query, but as a safeguard.
		}

		taskRecords, ok := recordsByTaskID[task.ID]
		if !ok || len(taskRecords) == 0 {
			log.Printf("No records found in pre-fetched map for task %d. Skipping.", task.ID)
			continue
		}

		// --- Refactored Tracking Logic ---

		// 5.1. Identify all unique accounts involved in this task.
		uniqueAccounts := make(map[int64]*model.Account)
		sendersByAccount := make(map[int64][]*model.AccountSender)

		for _, record := range taskRecords {
			if sender, senderOK := sendersByID[record.AccountSenderID]; senderOK {
				if _, accOK := uniqueAccounts[sender.AccountID]; !accOK {
					uniqueAccounts[sender.AccountID] = &sender.Account
				}
				sendersByAccount[sender.AccountID] = append(sendersByAccount[sender.AccountID], sender)
			}
		}

		// 5.2. Initialize aggregate counters for the task.
		newTotalOpenCount := 0
		newTotalClickCount := 0
		newTotalUniqueOpenCount := 0
		newTotalUniqueClickCount := 0

		// 5.3. Loop through each unique account, fetch its data, and aggregate.
		for accountID, account := range uniqueAccounts {
			// Find a valid sender email for this account to use in the API call.
			accountSenders := sendersByAccount[accountID]
			if len(accountSenders) == 0 {
				log.Printf("Warning: No senders found for account %d in task %d. Skipping tracking for this account.", accountID, task.ID)
				continue
			}
			representativeSenderEmail := accountSenders[0].EmailAddress

			// Create client for this specific account.
			client, err := aliyun.NewClient(s.aliyunEndpoint, account.AccessKeyID, account.AccessKeySecret)
			if err != nil {
				log.Printf("Error creating Aliyun client for account %d in task %d: %v", accountID, task.ID, err)
				continue
			}
			aliyunSender := aliyun.NewEmailSender(client)

			startTime := task.CreatedAt.Format("2006-01-02")
			endTime := time.Now().Format("2006-01-02")

			responseBody, err := aliyunSender.GetTrackList(representativeSenderEmail, *task.AliyunTagName, startTime, endTime)
			if err != nil {
				log.Printf("Error getting track list for task %d, account %d (tag: %s): %v", task.ID, accountID, *task.AliyunTagName, err)
				continue // Don't let one account's failure stop the whole sync.
			}

			if responseBody != nil && responseBody.Data != nil && responseBody.Data.Stat != nil && len(responseBody.Data.Stat) > 0 {
				stat := responseBody.Data.Stat[0]
				openCount, _ := strconv.Atoi(*stat.RcptOpenCount)
				clickCount, _ := strconv.Atoi(*stat.RcptClickCount)
				uniqueOpenCount, _ := strconv.Atoi(*stat.RcptUniqueOpenCount)
				uniqueClickCount, _ := strconv.Atoi(*stat.RcptUniqueClickCount)

				newTotalOpenCount += openCount
				newTotalClickCount += clickCount
				newTotalUniqueOpenCount += uniqueOpenCount
				newTotalUniqueClickCount += uniqueClickCount
			}
		}

		// 5.4. Update the task with the new aggregated totals and calculated rates.
		task.OpenCount = newTotalOpenCount
		task.ClickCount = newTotalClickCount
		task.UniqueOpenCount = newTotalUniqueOpenCount
		task.UniqueClickCount = newTotalUniqueClickCount

		if task.TotalRecipients > 0 {
			task.OpenRate = float64(task.OpenCount) / float64(task.TotalRecipients)
			task.ClickRate = float64(task.ClickCount) / float64(task.TotalRecipients)
			task.UniqueOpenRate = float64(task.UniqueOpenCount) / float64(task.TotalRecipients)
			task.UniqueClickRate = float64(task.UniqueClickCount) / float64(task.TotalRecipients)
		}

		if err := s.taskRepo.Update(task); err != nil {
			log.Printf("Error updating task %d with aggregated tracking data: %v", task.ID, err)
		} else {
			log.Printf("Successfully updated task %d with aggregated tracking data: Opens=%d, Clicks=%d", task.ID, task.OpenCount, task.ClickCount)
		}
		// --- End Refactored Logic ---

		// Add a delay between each task processing to avoid hitting API rate limits.
		time.Sleep(200 * time.Millisecond) // 200ms delay
	}

	return nil
}

// SyncSentCountStatistics aggregates sent counts from local records and updates the statistics table.
// This function is now less critical for task-level accuracy but still useful for sender-level daily stats.
func (s *TrackingService) SyncSentCountStatistics(ctx context.Context) error {
	// Aggregate stats for the last 3 days to catch any delays or missed updates.
	since := time.Now().Add(-72 * time.Hour)
	results, err := s.recordRepo.GetAggregatedSentCounts(since)
	if err != nil {
		return fmt.Errorf("failed to get aggregated sent counts: %w", err)
	}

	if len(results) == 0 {
		return nil
	}

	log.Printf("Found %d daily sender stats to update since %v", len(results), since.Format("2006-01-02"))

	// To perform updates, we need the account ID for each sender.
	senderIDs := make([]int64, 0, len(results))
	for _, res := range results {
		senderIDs = append(senderIDs, res.AccountSenderID)
	}
	senders, err := s.senderRepo.FindAccountSenderDetailsByIDs(senderIDs)
	if err != nil {
		return fmt.Errorf("failed to get sender details for stats update: %w", err)
	}
	senderAccountMap := make(map[int64]int64)
	for _, sender := range senders {
		senderAccountMap[sender.ID] = sender.AccountID
	}

	for _, result := range results {
		accountID, ok := senderAccountMap[result.AccountSenderID]
		if !ok {
			log.Printf("Warning: Could not find account ID for sender %d. Skipping stat update.", result.AccountSenderID)
			continue
		}

		stat := &model.SendStatistics{
			StatDate:        result.StatDate,
			AccountID:       accountID,
			AccountSenderID: result.AccountSenderID,
			SentCount:       result.SentCount,
			// Open/Click counts are handled by the other sync method
		}

		if err := s.statsRepo.CreateOrUpdate(stat); err != nil {
			log.Printf("Error updating sent count statistics for sender %d on %s: %v",
				result.AccountSenderID, result.StatDate.Format("2006-01-02"), err)
		}
	}

	return nil
}

func parseFloatRate(rateStr string) (float64, error) {
	if strings.HasSuffix(rateStr, "%") {
		rateStr = strings.TrimSuffix(rateStr, "%")
		val, err := strconv.ParseFloat(rateStr, 64)
		if err != nil {
			return 0, err
		}
		return val / 100.0, nil
	}
	return strconv.ParseFloat(rateStr, 64)
}
