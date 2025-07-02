package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"time"

	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
)

// TaskDispatcherService is responsible for dequeuing batch tasks and
// enqueuing individual email sending jobs.
type TaskDispatcherService struct {
	taskRepo     repository.EmailTaskRepository
	recordRepo   repository.EmailSendRecordRepository
	templateRepo repository.TemplateRepository
	queue        queue.QueueService
}

// NewTaskDispatcherService creates a new TaskDispatcherService.
func NewTaskDispatcherService(
	taskRepo repository.EmailTaskRepository,
	recordRepo repository.EmailSendRecordRepository,
	templateRepo repository.TemplateRepository,
	queueSvc queue.QueueService,
) *TaskDispatcherService {
	return &TaskDispatcherService{
		taskRepo:     taskRepo,
		recordRepo:   recordRepo,
		templateRepo: templateRepo,
		queue:        queueSvc,
	}
}

// Start begins the background worker loop for dispatching tasks.
func (s *TaskDispatcherService) Start(ctx context.Context) {
	log.Println("Starting task dispatcher service...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Task dispatcher service shutting down.")
				return
			default:
				s.processTask(ctx)
			}
		}
	}()
}

// processTask dequeues and processes a single task.
func (s *TaskDispatcherService) processTask(ctx context.Context) {
	log.Println("Dispatcher waiting for new task...")
	// Dequeue a task ID from the 'tasks:created' queue
	taskIDStr, err := s.queue.Dequeue(ctx, queue.TaskCreatedQueue)
	if err != nil {
		log.Printf("Error dequeuing task: %v", err)
		// On error (e.g., Redis connection issue), wait before retrying
		time.Sleep(5 * time.Second)
		return
	}
	log.Printf("Dispatcher processing task ID: %s", taskIDStr)

	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		log.Printf("Error parsing task ID '%s': %v", taskIDStr, err)
		return
	}

	// Fetch the full task details, including recipients
	task, err := s.taskRepo.FindByID(uint(taskID))
	if err != nil {
		log.Printf("Error finding task ID %d: %v", taskID, err)
		return
	}

	var subjectTpl, bodyTpl string
	// Pre-load template content if the task uses a template to avoid N+1 query problem.
	if task.TemplateID != nil {
		template, err := s.templateRepo.FindByID(*task.TemplateID)
		if err != nil {
			log.Printf("Failed to load template ID %d for task %d, aborting dispatch: %v", *task.TemplateID, task.ID, err)
			return // Abort processing this task if template is not found
		}
		subjectTpl = template.Subject
		bodyTpl = template.Body
	}

	// For each recipient, create a send record and enqueue it.
	for _, recipient := range task.Recipients {
		// If not using a template, get content directly from the task.
		// This part is inside the loop in case we want to support per-recipient task overrides in the future.
		if task.TemplateID == nil {
			// Prepare subject and body from task, handling nil pointers
			if task.Subject != nil {
				subjectTpl = *task.Subject
			}
			if task.Body != nil {
				bodyTpl = *task.Body
			}
		}

		// Render the templates with the recipient's metadata
		renderedSubject, renderedBody, err := s.renderTemplates(subjectTpl, bodyTpl, recipient.Metadata)
		if err != nil {
			log.Printf("Error rendering template for recipient %s (Task %d): %v", recipient.Email, task.ID, err)
			continue // Skip to next recipient
		}

		// 1. Create the email send record
		record := &model.EmailSendRecord{
			TaskID:          &task.ID,
			AccountSenderID: task.AccountSenderID,
			RecipientEmail:  recipient.Email,
			Subject:         renderedSubject,
			Body:            renderedBody,
			Status:          model.RecordStatusPending,
		}

		if err := s.recordRepo.Create(record); err != nil {
			log.Printf("Error creating send record for task %d, recipient %s: %v", task.ID, recipient.Email, err)
			continue // Skip to the next recipient
		}

		// 2. Enqueue the record ID for the email workers
		recordIDStr := strconv.FormatUint(uint64(record.ID), 10)
		if err := s.queue.Enqueue(ctx, queue.EmailSendingQueue, recordIDStr); err != nil {
			log.Printf("Error enqueuing send record ID %d for task %d: %v", record.ID, task.ID, err)
			// This is more critical, maybe implement a retry later
		}
	}

	log.Printf("Successfully dispatched all %d emails for task ID: %d", len(task.Recipients), task.ID)
}

func (s *TaskDispatcherService) renderTemplates(subjectTpl, bodyTpl string, metadata json.RawMessage) (string, string, error) {
	// Unmarshal metadata into a map
	var data map[string]interface{}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &data); err != nil {
			return "", "", fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		// Ensure data is not nil to avoid panics in template execution
		data = make(map[string]interface{})
	}

	// Render Subject
	subjectTmpl, err := template.New("subject").Parse(subjectTpl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse subject template: %w", err)
	}
	var subjectBuf bytes.Buffer
	if err := subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	// Render Body
	bodyTmpl, err := template.New("body").Parse(bodyTpl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse body template: %w", err)
	}
	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute body template: %w", err)
	}

	return subjectBuf.String(), bodyBuf.String(), nil
}
