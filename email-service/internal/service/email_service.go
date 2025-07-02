package service

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"email-service/pkg/aliyun"
	"fmt"
	"strconv"
	"time"
)

// EmailService handles the business logic for sending emails.
type EmailService struct {
	senderRepo     repository.SenderRepository
	recordRepo     repository.EmailSendRecordRepository
	queueService   queue.QueueService
	aliyunEndpoint string
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	senderRepo repository.SenderRepository,
	recordRepo repository.EmailSendRecordRepository,
	queueService queue.QueueService,
	aliyunEndpoint string,
) *EmailService {
	return &EmailService{
		senderRepo:     senderRepo,
		recordRepo:     recordRepo,
		queueService:   queueService,
		aliyunEndpoint: aliyunEndpoint,
	}
}

// QueueEmailForSending creates a pending email record and queues its ID for sending.
func (s *EmailService) QueueEmailForSending(ctx context.Context, accountSenderID int64, toAddress, subject, htmlBody string) (*model.EmailSendRecord, error) {
	// 1. Create the initial send record with "pending" status and all email details.
	record := &model.EmailSendRecord{
		AccountSenderID: accountSenderID,
		RecipientEmail:  toAddress,
		Subject:         subject,  // Save subject
		Body:            htmlBody, // Save body
		Status:          "pending",
	}

	if err := s.recordRepo.Create(record); err != nil {
		return nil, fmt.Errorf("failed to create send record: %w", err)
	}

	// 2. Enqueue the record ID for the background worker to process.
	recordIDStr := strconv.FormatInt(record.ID, 10)
	err := s.queueService.Enqueue(ctx, "email:sending", recordIDStr)
	if err != nil {
		// If queuing fails, mark the record as failed to avoid it being stuck in pending.
		wrappedErr := fmt.Errorf("failed to enqueue email for sending: %w", err)
		errMsg := wrappedErr.Error()
		record.Status = "failed"
		record.ErrorMessage = &errMsg
		_ = s.recordRepo.Update(record) // Best effort to update status
		return record, wrappedErr
	}

	return record, nil
}

// ProcessQueuedEmail fetches a record by ID and sends the email.
// This function is designed to be called by the background worker.
func (s *EmailService) ProcessQueuedEmail(recordID int64) error {
	// 1. Fetch the full record from the database.
	record, err := s.recordRepo.FindByID(uint(recordID))
	if err != nil {
		return fmt.Errorf("worker: failed to find record with id %d: %w", recordID, err)
	}

	// 2. Fetch complete sender details, including Account credentials.
	accountSender, err := s.senderRepo.FindAccountSenderDetails(record.AccountSenderID)
	if err != nil {
		// Mark record as failed if sender details can't be found.
		wrappedErr := fmt.Errorf("worker: failed to find account sender details for id %d: %w", record.AccountSenderID, err)
		errMsg := wrappedErr.Error()
		record.Status = "failed"
		record.ErrorMessage = &errMsg
		_ = s.recordRepo.Update(record)
		return wrappedErr
	}

	// 3. Mark the record as "processing".
	record.Status = "processing"
	if err := s.recordRepo.Update(record); err != nil {
		// If we can't update to "processing", log it but proceed.
		fmt.Printf("worker: failed to update record %d status to 'processing': %v\n", recordID, err)
	}

	// 4. Create a temporary Aliyun client.
	client, err := aliyun.NewClient(
		s.aliyunEndpoint,
		accountSender.Account.AccessKeyID,
		accountSender.Account.AccessKeySecret,
	)
	if err != nil {
		wrappedErr := fmt.Errorf("worker: failed to create aliyun client: %w", err)
		errMsg := wrappedErr.Error()
		record.Status = "failed"
		record.ErrorMessage = &errMsg
		_ = s.recordRepo.Update(record)
		return wrappedErr
	}

	// 5. Send email via Aliyun.
	aliyunSender := aliyun.NewEmailSender(client)
	err = aliyunSender.SendSingleEmail(
		accountSender.EmailAddress,
		accountSender.Sender.Name,
		record.RecipientEmail,
		record.Subject,
		record.Body,
	)

	// 6. Update the record based on the sending result.
	if err != nil {
		errMsg := err.Error()
		record.Status = "failed"
		record.ErrorMessage = &errMsg
	} else {
		record.Status = "sent"
		now := time.Now()
		record.SentAt = &now
		record.LastStatusUpdateAt = &now
	}

	if updateErr := s.recordRepo.Update(record); updateErr != nil {
		return fmt.Errorf("worker: failed to update final status for record %d: %w", recordID, updateErr)
	}

	return nil
}
