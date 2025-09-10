package service

import (
	"context"
	"email-service/internal/queue"
	"fmt"
	"log"
	"strconv"
	"time"
)

// RecipientImportWorkerService defines the interface for the import worker.
type RecipientImportWorkerService interface {
	StartWorker(ctx context.Context) error
	ProcessImportJob(taskID string) error
}

type recipientImportWorkerService struct {
	queueService  queue.QueueService
	importService RecipientImportService
	workerID      string
}

// NewRecipientImportWorkerService creates a new RecipientImportWorkerService.
func NewRecipientImportWorkerService(
	queueService queue.QueueService,
	importService RecipientImportService,
) RecipientImportWorkerService {
	return &recipientImportWorkerService{
		queueService:  queueService,
		importService: importService,
		workerID:      fmt.Sprintf("import-worker-%d", time.Now().UnixNano()),
	}
}

func (w *recipientImportWorkerService) StartWorker(ctx context.Context) error {
	log.Printf("Starting recipient import worker: %s", w.workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping recipient import worker: %s", w.workerID)
			return ctx.Err()
		default:
			// Try to dequeue a message
			taskID, err := w.queueService.Dequeue(ctx, queue.RecipientImportQueue)
			if err != nil {
				log.Printf("Worker %s: Failed to dequeue from import queue: %v", w.workerID, err)
				// Sleep briefly before retrying to avoid busy waiting
				time.Sleep(5 * time.Second)
				continue
			}

			if taskID == "" {
				// No message available, sleep briefly before trying again
				time.Sleep(1 * time.Second)
				continue
			}

			// Process the import job
			if err := w.ProcessImportJob(taskID); err != nil {
				log.Printf("Worker %s: Failed to process import job %s: %v", w.workerID, taskID, err)
			} else {
				log.Printf("Worker %s: Successfully processed import job %s", w.workerID, taskID)
			}
		}
	}
}

func (w *recipientImportWorkerService) ProcessImportJob(taskID string) error {
	log.Printf("Worker %s: Processing import job %s", w.workerID, taskID)

	// Convert taskID to int64
	taskIDInt, err := strconv.ParseInt(taskID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid task ID format: %s", taskID)
	}

	// Process the import task
	if err := w.importService.ProcessImportTask(taskIDInt); err != nil {
		return fmt.Errorf("failed to process import task %d: %w", taskIDInt, err)
	}

	log.Printf("Worker %s: Completed processing import job %s", w.workerID, taskID)
	return nil
}
