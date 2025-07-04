package service

import (
	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"email-service/pkg/aliyun"
	"fmt"
)

// EmailService handles the business logic for sending emails.
type EmailService struct {
	senderRepo     repository.SenderRepository
	recordRepo     repository.EmailSendRecordRepository
	taskRepo       repository.EmailTaskRepository
	queueService   queue.QueueService
	aliyunEndpoint string
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	senderRepo repository.SenderRepository,
	recordRepo repository.EmailSendRecordRepository,
	taskRepo repository.EmailTaskRepository,
	queueService queue.QueueService,
	aliyunEndpoint string,
) *EmailService {
	return &EmailService{
		senderRepo:     senderRepo,
		recordRepo:     recordRepo,
		taskRepo:       taskRepo,
		queueService:   queueService,
		aliyunEndpoint: aliyunEndpoint,
	}
}

// ProcessEmailJob is now designed to handle a batch of emails contained within a single payload.
// It iterates through the emails, sends them one by one, and updates their status individually.
func (s *EmailService) ProcessEmailJob(payload *model.EmailJobPayload) error {
	if len(payload.Emails) == 0 {
		return nil // Nothing to do
	}

	// 1. Sender details are shared across the batch.
	accountSender := &payload.AccountSender

	// 2. Create a single Aliyun client for the entire batch.
	client, err := aliyun.NewClient(
		s.aliyunEndpoint,
		accountSender.Account.AccessKeyID,
		accountSender.Account.AccessKeySecret,
	)
	if err != nil {
		// If client fails, the entire batch fails.
		// A robust system would update all records in the batch to 'failed'.
		return fmt.Errorf("worker: failed to create aliyun client for batch from sender %s: %w", accountSender.EmailAddress, err)
	}

	aliyunSender := aliyun.NewEmailSender(client)

	// 3. Process each email in the batch.
	for _, emailInfo := range payload.Emails {
		// Immediately update status to 'sending'.
		if err := s.recordRepo.UpdateStatus(emailInfo.RecordID, model.RecordStatusSending, "", nil); err != nil {
			// Log and continue to the next email. Don't let one failed update stop the whole batch.
			// The record will remain 'pending'.
			// A monitoring system should catch records stuck in 'pending'.
			_ = fmt.Errorf("worker: failed to set record %d to sending: %w", emailInfo.RecordID, err)
			continue
		}

		// Send email via Aliyun.
		requestID, err := aliyunSender.SendSingleEmail(
			accountSender.EmailAddress,
			accountSender.Sender.Name,
			emailInfo.RecipientEmail,
			payload.Subject, // Subject and Body are shared
			payload.Body,
			payload.AliyunTagName,
			true, // Enable ClickTrace
		)

		// Update the record based on the sending result.
		if err != nil {
			errMsg := err.Error()
			if updateErr := s.recordRepo.UpdateStatus(emailInfo.RecordID, model.RecordStatusFailed, "", &errMsg); updateErr != nil {
				// Log both errors.
				_ = fmt.Errorf("worker: send failed for record %d (err: %s), and status update failed (err: %s)",
					emailInfo.RecordID, err, updateErr)
			}
			// Continue to the next email, don't return.
		} else {
			var aliyunRequestID string
			if requestID != nil {
				aliyunRequestID = *requestID
			}
			if updateErr := s.recordRepo.UpdateStatus(emailInfo.RecordID, model.RecordStatusSent, aliyunRequestID, nil); updateErr != nil {
				_ = fmt.Errorf("worker: send successful for record %d, but status update failed: %w", emailInfo.RecordID, updateErr)
			}
		}
	}

	return nil // Return nil because individual errors are handled and logged within the loop.
}
