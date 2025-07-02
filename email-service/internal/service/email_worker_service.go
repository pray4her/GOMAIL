package service

import (
	"context"
	"email-service/internal/queue"
	"log"
	"strconv"
	"time"
)

// EmailProcessor defines the interface for processing a single email task.
// This decouples the worker from the full EmailService.
type EmailProcessor interface {
	ProcessQueuedEmail(recordID int64) error
}

// EmailWorkerService is the background worker that processes email sending tasks.
type EmailWorkerService struct {
	queueService   queue.QueueService
	emailProcessor EmailProcessor
}

// NewEmailWorkerService creates a new worker service.
func NewEmailWorkerService(qs queue.QueueService, ep EmailProcessor) *EmailWorkerService {
	return &EmailWorkerService{
		queueService:   qs,
		emailProcessor: ep,
	}
}

// Start begins the worker loop with a specified number of concurrent workers.
func (w *EmailWorkerService) Start(ctx context.Context, workerCount int) {
	log.Printf("Starting email worker service with %d workers...", workerCount)

	for i := 0; i < workerCount; i++ {
		workerID := i + 1
		go func() {
			log.Printf("Worker %d started.", workerID)
			for {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d shutting down.", workerID)
					return
				default:
					w.processTask(ctx, workerID)
				}
			}
		}()
	}
}

func (w *EmailWorkerService) processTask(ctx context.Context, workerID int) {
	// Dequeue a task. This will block until a message is available.
	message, err := w.queueService.Dequeue(ctx, queue.EmailSendingQueue)
	if err != nil {
		// If context is cancelled, Dequeue will return an error.
		// We can log other errors but should be careful not to crash the worker.
		log.Printf("[Worker %d] Error dequeuing email task: %v. Retrying in 5 seconds...", workerID, err)
		time.Sleep(5 * time.Second)
		return
	}

	recordID, err := strconv.ParseInt(message, 10, 64)
	if err != nil {
		log.Printf("[Worker %d] Error parsing record ID from message '%s': %v", workerID, message, err)
		// Don't re-queue, as the message is malformed.
		return
	}

	log.Printf("[Worker %d] Processing email task for record ID: %d", workerID, recordID)

	// Process the email. This is where the actual sending happens.
	if err := w.emailProcessor.ProcessQueuedEmail(recordID); err != nil {
		log.Printf("[Worker %d] Error processing email for record ID %d: %v", workerID, recordID, err)
		// Implement retry logic or move to a dead-letter queue if necessary.
		// For now, we just log the error.
	} else {
		log.Printf("[Worker %d] Successfully processed email task for record ID: %d", workerID, recordID)
	}
}
