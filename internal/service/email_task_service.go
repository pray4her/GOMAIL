package service

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

var (
	ErrPermissionDenied = errors.New("user does not have permission to use this sender")
	ErrValidation       = errors.New("task validation failed")
)

// We define an interface for the parts of RecipientRepository we need.
// This makes the service easier to test.
type RecipientFinder interface {
	FindByIds(ids []uint) ([]model.Recipient, error)
}

// EmailTaskService handles the business logic for creating and managing email tasks.
type EmailTaskService struct {
	taskRepo     repository.EmailTaskRepository
	groupRepo    repository.RecipientGroupRepository
	templateRepo repository.TemplateRepository
	queue        queue.QueueService
}

// NewEmailTaskService creates a new EmailTaskService.
func NewEmailTaskService(
	taskRepo repository.EmailTaskRepository,
	groupRepo repository.RecipientGroupRepository,
	templateRepo repository.TemplateRepository,
	queueSvc queue.QueueService,
) *EmailTaskService {
	return &EmailTaskService{
		taskRepo:     taskRepo,
		groupRepo:    groupRepo,
		templateRepo: templateRepo,
		queue:        queueSvc,
	}
}

// CreateEmailTask creates a task from either a template or raw content,
// validates recipients, and queues the task for dispatching.
func (s *EmailTaskService) CreateEmailTask(
	ctx context.Context,
	userID int64,
	taskName string,
	recipientGroupID int64,
	templateID *int64,
	subject, body string,
	scheduledAt *time.Time,
) (*model.EmailTask, error) {

	// --- Validation ---
	isTemplateTask := templateID != nil
	isDirectTask := subject != "" && body != ""

	if isTemplateTask && isDirectTask {
		return nil, fmt.Errorf("%w: cannot provide both template_id and subject/body", ErrValidation)
	}
	if !isTemplateTask && !isDirectTask {
		return nil, fmt.Errorf("%w: must provide either template_id or both subject and body", ErrValidation)
	}
	if isTemplateTask {
		// Verify the template exists
		if _, err := s.templateRepo.FindByID(*templateID); err != nil {
			return nil, fmt.Errorf("%w: template with id %d not found", ErrValidation, *templateID)
		}
	}

	// 1. Validate the recipient group
	if _, err := s.groupRepo.FindByID(recipientGroupID); err != nil {
		return nil, fmt.Errorf("%w: recipient group with id %d not found", ErrValidation, recipientGroupID)
	}

	// 2. Create the task object
	task := &model.EmailTask{
		TaskName:         taskName,
		Status:           "pending",
		CreatedByUserID:  userID,
		ScheduledAt:      scheduledAt,
		RecipientGroupID: &recipientGroupID,
	}

	if isTemplateTask {
		task.TemplateID = templateID
	} else {
		task.Subject = &subject
		task.Body = &body
	}

	// 3. Save the task to the database
	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create email task: %w", err)
	}

	// 4. Enqueue the task ID for processing
	taskIDStr := strconv.FormatInt(task.ID, 10)

	// --- Scheduling Logic ---
	if scheduledAt != nil && scheduledAt.After(time.Now()) {
		// This is a future task, add it to the scheduled queue (sorted set)
		task.Status = "scheduled"
		if err := s.taskRepo.Update(task); err != nil {
			return task, fmt.Errorf("failed to update task status to scheduled: %w", err)
		}

		err := s.queue.EnqueueScheduled(ctx, queue.TaskScheduledQueue, taskIDStr, float64(scheduledAt.Unix()))
		if err != nil {
			return task, fmt.Errorf("task created (ID: %d) but failed to enqueue into scheduled queue: %w", task.ID, err)
		}
		log.Printf("Task %d scheduled for future execution at %v", task.ID, *scheduledAt)
	} else {
		// This is an immediate task, enqueue it for the dispatcher service
		err := s.queue.Enqueue(ctx, queue.TaskCreatedQueue, taskIDStr)
		if err != nil {
			// If queuing fails, we have an orphaned task.
			// A robust system would have a recovery mechanism (e.g., a background job).
			// For now, we return an error and log the issue.
			return task, fmt.Errorf("task created (ID: %d) but failed to enqueue for dispatch: %w", task.ID, err)
		}
	}
	// --- End Scheduling Logic ---

	return task, nil
}
