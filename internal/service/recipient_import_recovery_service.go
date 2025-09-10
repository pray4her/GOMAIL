package service

import (
	"context"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"fmt"
	"log"
	"strconv"
)

// RecipientImportRecoveryService handles recovery of import task states after system restart.
type RecipientImportRecoveryService interface {
	RecoverInProgressImportTasks() error
	RequeueImportTask(taskID int64) error
}

type recipientImportRecoveryService struct {
	importRepo   repository.RecipientImportTaskRepository
	queueService queue.QueueService
}

// NewRecipientImportRecoveryService creates a new RecipientImportRecoveryService instance.
func NewRecipientImportRecoveryService(
	importRepo repository.RecipientImportTaskRepository,
	queueService queue.QueueService,
) RecipientImportRecoveryService {
	return &recipientImportRecoveryService{
		importRepo:   importRepo,
		queueService: queueService,
	}
}

// RecoverInProgressImportTasks scans the database for import tasks that were in progress when the system shut down
// and re-queues them for processing.
func (s *recipientImportRecoveryService) RecoverInProgressImportTasks() error {
	log.Println("Starting recipient import task recovery process...")

	// Find all import tasks that were in progress when the system shut down
	tasks, err := s.importRepo.FindInProgressTasks()
	if err != nil {
		return fmt.Errorf("failed to find in-progress import tasks: %w", err)
	}

	if len(tasks) == 0 {
		log.Println("No in-progress import tasks found. Recovery complete.")
		return nil
	}

	log.Printf("Found %d import tasks to recover", len(tasks))

	recoveredCount := 0
	for _, task := range tasks {
		if err := s.RequeueImportTask(task.ID); err != nil {
			log.Printf("Failed to recover import task %d: %v", task.ID, err)
			continue
		}
		recoveredCount++
	}

	log.Printf("Import task recovery complete. Successfully recovered %d out of %d tasks.", recoveredCount, len(tasks))
	return nil
}

// RequeueImportTask re-queues a specific import task for processing.
func (s *recipientImportRecoveryService) RequeueImportTask(taskID int64) error {
	log.Printf("Re-queuing import task %d for recovery", taskID)

	// Enqueue the task for processing
	if err := s.queueService.Enqueue(context.Background(), queue.RecipientImportQueue, strconv.FormatInt(taskID, 10)); err != nil {
		return fmt.Errorf("failed to enqueue import task %d for recovery: %w", taskID, err)
	}

	log.Printf("Successfully re-queued import task %d", taskID)
	return nil
}
