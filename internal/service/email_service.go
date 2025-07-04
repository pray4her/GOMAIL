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

// ProcessEmailJob fetches a record by ID and sends the email.
// This function is designed to be called by the background worker.
// It receives all necessary data in the payload to avoid DB queries.
func (s *EmailService) ProcessEmailJob(payload *model.EmailJobPayload) error {
	// 1. Immediately update status to 'sending'. This also verifies the record exists.
	if err := s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusSending, "", nil); err != nil {
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
		_ = s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusFailed, "", &errMsg)
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
		true, // Enable ClickTrace
	)

	// 5. Update the record based on the sending result.
	if err != nil {
		errMsg := err.Error()
		if updateErr := s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusFailed, "", &errMsg); updateErr != nil {
			return fmt.Errorf("worker: failed to update final status for record %d: %w", payload.RecordID, updateErr)
		}
		return err // Return the original sending error
	}

	var aliyunRequestID string
	if requestID != nil {
		aliyunRequestID = *requestID
	}

	if updateErr := s.recordRepo.UpdateStatus(payload.RecordID, model.RecordStatusSent, aliyunRequestID, nil); updateErr != nil {
		return fmt.Errorf("worker: failed to update final status for record %d: %w", payload.RecordID, updateErr)
	}

	return nil
}
