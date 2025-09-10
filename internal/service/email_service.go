package service

import (
	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"email-service/pkg/aliyun"
	"fmt"
	"log"
)

// EmailService handles the business logic for sending emails.
type EmailService struct {
	senderRepo     repository.SenderRepository
	recordRepo     repository.EmailSendRecordRepository
	taskRepo       repository.EmailTaskRepository
	counterService TaskCounterService
	queueService   queue.QueueService
	aliyunEndpoint string
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	senderRepo repository.SenderRepository,
	recordRepo repository.EmailSendRecordRepository,
	taskRepo repository.EmailTaskRepository,
	counterService TaskCounterService,
	queueService queue.QueueService,
	aliyunEndpoint string,
) *EmailService {
	return &EmailService{
		senderRepo:     senderRepo,
		recordRepo:     recordRepo,
		taskRepo:       taskRepo,
		counterService: counterService,
		queueService:   queueService,
		aliyunEndpoint: aliyunEndpoint,
	}
}

// ProcessEmailJob fetches a record by ID and sends the email.
// This function is designed to be called by the background worker.
// It receives all necessary data in the payload to avoid DB queries.
func (s *EmailService) ProcessEmailJob(payload *model.EmailJobPayload) error {
	// 1. Immediately update status to 'sending'. This also verifies the record exists.
	if err := s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusSending, nil, nil); err != nil {
		return fmt.Errorf("worker: failed to set record %d to sending: %w", payload.RecordID, err)
	}

	// 2. Sender details are already in the payload.
	accountSender := &payload.AccountSender

	// 3. Create a temporary Aliyun client.
	client, err := aliyun.NewClient(
		s.aliyunEndpoint,
		accountSender.Account.AccessKeyID,
		accountSender.Account.AccessKeySecret,
	)
	if err != nil {
		wrappedErr := fmt.Errorf("worker: failed to create aliyun client: %w", err)
		errMsg := wrappedErr.Error()
		// Best effort to update status, return the creation error anyway.
		_ = s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusFailed, nil, &errMsg)
		return wrappedErr
	}

	// 4. Send email via Aliyun. Tag name is already in the payload.
	aliyunSender := aliyun.NewEmailSender(client)
	requestID, err := aliyunSender.SendSingleEmail(
		accountSender.EmailAddress,
		accountSender.Sender.Name,
		payload.RecipientEmail,
		payload.Subject,
		payload.Body,
		payload.AliyunTagName,
		accountSender.ReplyToEmail, // 回信地址
		true,                       // Enable ClickTrace
	)

	// 5. Update the record based on the sending result.
	if err != nil {
		errMsg := err.Error()
		if updateErr := s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusFailed, nil, &errMsg); updateErr != nil {
			return fmt.Errorf("worker: failed to update final status for record %d: %w", payload.RecordID, updateErr)
		}
		// Decrement counter and check for task completion
		s.handleCounterDecrement(payload.TaskID)
		return err // Return the original sending error
	}

	var aliyunRequestID string
	if requestID != nil {
		aliyunRequestID = *requestID
	}

	// Update the record status to "sent"
	if err := s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusSent, &aliyunRequestID, nil); err != nil {
		// Even if this fails, the email was sent. Log it and move on.
		log.Printf("worker: CRITICAL: failed to update record %d status to sent: %v", payload.RecordID, err)
	}

	// Decrement counter and check for task completion
	s.handleCounterDecrement(payload.TaskID)

	return nil
}

// handleCounterDecrement decrements the task counter and handles completion if needed.
func (s *EmailService) handleCounterDecrement(taskID int64) {
	remaining, err := s.counterService.DecrementTaskCounter(taskID)
	if err != nil {
		// Log error but don't fail the email sending
		fmt.Printf("Warning: Failed to decrement counter for task %d: %v\n", taskID, err)
		return
	}

	// If counter reached zero, handle task completion
	if remaining == 0 {
		if err := s.handleTaskCompletion(taskID); err != nil {
			fmt.Printf("Error handling task completion for task %d: %v\n", taskID, err)
		}
	}
}

// handleTaskCompletion marks a task as completed and cleans up resources.
func (s *EmailService) handleTaskCompletion(taskID int64) error {
	// Update task status to completed
	if err := s.taskRepo.UpdateStatus(taskID, model.TaskStatusCompleted); err != nil {
		return fmt.Errorf("failed to update task %d status to completed: %w", taskID, err)
	}

	// Update progress statistics
	records, err := s.recordRepo.FindByTaskID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get records for task %d: %w", taskID, err)
	}

	sentCount := 0
	failedCount := 0
	for _, record := range records {
		switch record.Status {
		case model.RecordStatusSent:
			sentCount++
		case model.RecordStatusFailed:
			failedCount++
		}
	}

	if err := s.taskRepo.UpdateProgress(taskID, sentCount, failedCount); err != nil {
		return fmt.Errorf("failed to update task %d progress: %w", taskID, err)
	}

	// Clean up Redis counter
	if err := s.counterService.DeleteTaskCounter(taskID); err != nil {
		// Log warning but don't fail the completion
		fmt.Printf("Warning: Failed to delete counter for completed task %d: %v\n", taskID, err)
	}

	fmt.Printf("Task %d completed successfully. Sent: %d, Failed: %d\n", taskID, sentCount, failedCount)
	return nil
}
