package service

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"fmt"
	"strconv"
)

// We define an interface for the parts of RecipientRepository we need.
// This makes the service easier to test.
type RecipientFinder interface {
	FindByIds(ids []uint) ([]model.Recipient, error)
}

// EmailTaskService handles the business logic for creating and managing email tasks.
type EmailTaskService struct {
	taskRepo      repository.EmailTaskRepository
	recipientRepo RecipientFinder
	templateRepo  repository.TemplateRepository
	queue         queue.QueueService
}

// NewEmailTaskService creates a new EmailTaskService.
func NewEmailTaskService(
	taskRepo repository.EmailTaskRepository,
	recipientRepo RecipientFinder,
	templateRepo repository.TemplateRepository,
	queueSvc queue.QueueService,
) *EmailTaskService {
	return &EmailTaskService{
		taskRepo:      taskRepo,
		recipientRepo: recipientRepo,
		templateRepo:  templateRepo,
		queue:         queueSvc,
	}
}

// CreateBatchSendTask creates a task from either a template or raw content,
// validates recipients, and queues the task for dispatching.
func (s *EmailTaskService) CreateBatchSendTask(
	ctx context.Context,
	taskName string,
	accountSenderID int64,
	recipientIDs []uint,
	templateID *int64,
	subject, body string,
) (*model.EmailTask, error) {

	// --- Validation ---
	isTemplateTask := templateID != nil
	isDirectTask := subject != "" && body != ""

	if isTemplateTask && isDirectTask {
		return nil, fmt.Errorf("cannot provide both template_id and subject/body")
	}
	if !isTemplateTask && !isDirectTask {
		return nil, fmt.Errorf("must provide either template_id or both subject and body")
	}
	if isTemplateTask {
		// Verify the template exists
		if _, err := s.templateRepo.FindByID(*templateID); err != nil {
			return nil, fmt.Errorf("template with id %d not found: %w", *templateID, err)
		}
	}

	// 1. Validate recipients
	if len(recipientIDs) == 0 {
		return nil, fmt.Errorf("at least one recipient ID must be provided")
	}
	foundRecipients, err := s.recipientRepo.FindByIds(recipientIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to validate recipients: %w", err)
	}
	if len(foundRecipients) != len(recipientIDs) {
		return nil, fmt.Errorf("one or more recipient IDs are invalid")
	}

	// 2. Create the task object with associations
	task := &model.EmailTask{
		TaskName:        taskName,
		AccountSenderID: accountSenderID,
		Status:          "pending",
	}

	if isTemplateTask {
		task.TemplateID = templateID
	} else {
		task.Subject = &subject
		task.Body = &body
	}

	// GORM will automatically handle the many2many relationship
	for _, r := range foundRecipients {
		// We need to pass pointers to the array
		recipientCopy := r
		task.Recipients = append(task.Recipients, &recipientCopy)
	}

	// 3. Save the task and its associations to the database
	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("failed to create email task: %w", err)
	}

	// 4. Enqueue the task ID for the dispatcher service
	taskIDStr := strconv.FormatInt(task.ID, 10)
	if err := s.queue.Enqueue(ctx, queue.TaskCreatedQueue, taskIDStr); err != nil {
		// If queuing fails, we have an orphaned task.
		// A robust system would have a recovery mechanism (e.g., a background job).
		// For now, we return an error and log the issue.
		return task, fmt.Errorf("task created (ID: %d) but failed to enqueue for dispatch: %w", task.ID, err)
	}

	return task, nil
}
