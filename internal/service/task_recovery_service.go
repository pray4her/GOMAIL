package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
	"fmt"
	"log"
)

// TaskRecoveryService handles recovery of task states after system restart.
type TaskRecoveryService interface {
	RecoverInProgressTasks() error
	RebuildTaskCounter(taskID int64) error
}

type taskRecoveryService struct {
	taskRepo       repository.EmailTaskRepository
	recordRepo     repository.EmailSendRecordRepository
	counterService TaskCounterService
}

// NewTaskRecoveryService creates a new TaskRecoveryService instance.
func NewTaskRecoveryService(
	taskRepo repository.EmailTaskRepository,
	recordRepo repository.EmailSendRecordRepository,
	counterService TaskCounterService,
) TaskRecoveryService {
	return &taskRecoveryService{
		taskRepo:       taskRepo,
		recordRepo:     recordRepo,
		counterService: counterService,
	}
}

// RecoverInProgressTasks scans the database for tasks that were in progress when the system shut down
// and rebuilds their Redis counters.
func (s *taskRecoveryService) RecoverInProgressTasks() error {
	log.Println("Starting task recovery process...")

	// Find all tasks that were in progress when the system shut down
	tasks, err := s.taskRepo.FindInProgressTasks()
	if err != nil {
		return fmt.Errorf("failed to find in-progress tasks: %w", err)
	}

	if len(tasks) == 0 {
		log.Println("No in-progress tasks found. Recovery complete.")
		return nil
	}

	log.Printf("Found %d tasks to recover", len(tasks))

	recoveredCount := 0
	for _, task := range tasks {
		if err := s.RebuildTaskCounter(task.ID); err != nil {
			log.Printf("Failed to recover task %d: %v", task.ID, err)
			// Mark the task as failed if we can't recover it
			_ = s.taskRepo.UpdateStatus(task.ID, model.TaskStatusFailed)
			continue
		}
		recoveredCount++
	}

	log.Printf("Task recovery complete. Successfully recovered %d out of %d tasks.", recoveredCount, len(tasks))
	return nil
}

// RebuildTaskCounter rebuilds the Redis counter for a specific task based on database records.
func (s *taskRecoveryService) RebuildTaskCounter(taskID int64) error {
	// Get all records for this task
	records, err := s.recordRepo.FindByTaskID(taskID)
	if err != nil {
		return fmt.Errorf("failed to get records for task %d: %w", taskID, err)
	}

	if len(records) == 0 {
		// No records means the task never started dispatching properly
		// Mark it as failed
		log.Printf("Task %d has no records, marking as failed", taskID)
		return s.taskRepo.UpdateStatus(taskID, model.TaskStatusFailed)
	}

	// Count pending records (these are the ones that still need to be sent)
	pendingCount := 0
	sentCount := 0
	failedCount := 0

	for _, record := range records {
		switch record.Status {
		case model.RecordStatusPending, model.RecordStatusSending:
			pendingCount++
		case model.RecordStatusSent:
			sentCount++
		case model.RecordStatusFailed:
			failedCount++
		}
	}

	// If no pending records, the task is complete
	if pendingCount == 0 {
		log.Printf("Task %d has no pending records, marking as completed", taskID)
		if err := s.taskRepo.UpdateStatus(taskID, model.TaskStatusCompleted); err != nil {
			return fmt.Errorf("failed to mark task %d as completed: %w", taskID, err)
		}
		if err := s.taskRepo.UpdateProgress(taskID, sentCount, failedCount); err != nil {
			log.Printf("Warning: Failed to update progress for completed task %d: %v", taskID, err)
		}
		return nil
	}

	// Rebuild the Redis counter with the pending count
	if err := s.counterService.InitializeTaskCounter(taskID, pendingCount); err != nil {
		return fmt.Errorf("failed to rebuild counter for task %d: %w", taskID, err)
	}

	// Update the task progress
	if err := s.taskRepo.UpdateProgress(taskID, sentCount, failedCount); err != nil {
		log.Printf("Warning: Failed to update progress for task %d: %v", taskID, err)
	}

	log.Printf("Rebuilt counter for task %d: %d pending records", taskID, pendingCount)
	return nil
}
