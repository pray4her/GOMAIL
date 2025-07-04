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
	taskRepo        repository.EmailTaskRepository
	recordRepo      repository.EmailSendRecordRepository
	templateRepo    repository.TemplateRepository
	groupService    RecipientGroupService
	loadBalancer    LoadBalancerService
	senderRepo      repository.SenderRepository
	queue           queue.QueueService
	pollingInterval time.Duration
}

// NewTaskDispatcherService creates a new TaskDispatcherService.
func NewTaskDispatcherService(
	taskRepo repository.EmailTaskRepository,
	recordRepo repository.EmailSendRecordRepository,
	senderRepo repository.SenderRepository,
	templateRepo repository.TemplateRepository,
	groupService RecipientGroupService,
	loadBalancer LoadBalancerService,
	queueSvc queue.QueueService,
	pollingInterval time.Duration,
) *TaskDispatcherService {
	return &TaskDispatcherService{
		taskRepo:        taskRepo,
		recordRepo:      recordRepo,
		senderRepo:      senderRepo,
		templateRepo:    templateRepo,
		groupService:    groupService,
		loadBalancer:    loadBalancer,
		queue:           queueSvc,
		pollingInterval: pollingInterval,
	}
}

// Start begins the background worker loops for both immediate and scheduled tasks.
func (s *TaskDispatcherService) Start(ctx context.Context) {
	log.Println("Starting task dispatcher service...")

	// Start the worker for immediate tasks
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Immediate task dispatcher shutting down.")
				return
			default:
				s.processImmediateTask(ctx)
			}
		}
	}()

	// Start the scheduler for scheduled tasks
	go s.runScheduler(ctx)
}

// runScheduler is a loop that periodically checks for due scheduled tasks.
func (s *TaskDispatcherService) runScheduler(ctx context.Context) {
	log.Printf("Scheduler started, polling every %v", s.pollingInterval)
	ticker := time.NewTicker(s.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler shutting down.")
			return
		case <-ticker.C:
			s.processScheduledTasks(ctx)
		}
	}
}

// processScheduledTasks fetches due tasks from the scheduled queue and enqueues them for immediate processing.
func (s *TaskDispatcherService) processScheduledTasks(ctx context.Context) {
	now := time.Now().Unix()
	taskIDs, err := s.queue.DequeueDue(ctx, queue.TaskScheduledQueue, float64(now))
	if err != nil {
		log.Printf("Error dequeuing scheduled tasks: %v", err)
		return
	}

	if len(taskIDs) > 0 {
		log.Printf("Scheduler found %d due tasks. Re-queuing for immediate dispatch.", len(taskIDs))
		for _, taskID := range taskIDs {
			// Enqueue to the immediate queue for the other worker to pick up.
			// This reuses the entire existing dispatch logic.
			if err := s.queue.Enqueue(ctx, queue.TaskCreatedQueue, taskID); err != nil {
				log.Printf("CRITICAL: Failed to re-queue due task ID %s: %v", taskID, err)
				// A robust system would move this to a "failed" queue for manual inspection.
			}
		}
	}
}

// processImmediateTask dequeues and processes a single immediate task.
func (s *TaskDispatcherService) processImmediateTask(ctx context.Context) {
	log.Println("Dispatcher waiting for new immediate task...")
	taskIDStr, err := s.queue.Dequeue(ctx, queue.TaskCreatedQueue)
	if err != nil {
		log.Printf("Error dequeuing task: %v", err)
		time.Sleep(5 * time.Second)
		return
	}
	log.Printf("Dispatcher processing immediate task ID: %s", taskIDStr)

	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing task ID '%s': %v", taskIDStr, err)
		return
	}

	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		log.Printf("Error finding task ID %d: %v", taskID, err)
		return
	}

	if task.RecipientGroupID == nil {
		log.Printf("Task %d has no recipient group ID. Aborting dispatch.", task.ID)
		return
	}

	// === STREAMING REFACTOR: Process recipients in pages ===
	const pageSize = 10000 // Process 10,000 recipients per batch
	for page := 1; ; page++ {
		log.Printf("Processing page %d for task %d...", page, task.ID)

		recipients, err := s.groupService.ResolveRecipients(*task.RecipientGroupID, page, pageSize)
		if err != nil {
			log.Printf("CRITICAL: Failed to resolve recipients page %d for group %d (Task %d): %v. Aborting task.", page, *task.RecipientGroupID, task.ID, err)
			return // Abort on recipient resolution failure
		}
		if len(recipients) == 0 {
			log.Printf("Finished processing all pages for task %d.", task.ID)
			break // This was the last page
		}

		log.Printf("Page %d for task %d resolved to %d recipients. Generating dispatch plan...", page, task.ID, len(recipients))

		// === Load Balancing Logic for the current page ===
		plan, err := s.loadBalancer.GenerateDispatchPlan(task.CreatedByUserID, len(recipients))
		if err != nil {
			log.Printf("CRITICAL: Failed to generate dispatch plan for page %d of task %d: %v. Skipping page.", page, task.ID, err)
			continue // Skip to next page
		}

		if !plan.Possible || plan.TotalEmails == 0 {
			log.Printf("Could not dispatch page %d of task %d. Requested %d emails, but plan allows for %d. Insufficient quota or no active senders.", page, task.ID, len(recipients), plan.TotalEmails)
			continue // Skip to next page
		}
		log.Printf("Dispatch plan for page %d of task %d generated: %d emails to be sent using %d senders.", page, task.ID, plan.TotalEmails, len(plan.Assignments))

		// Step 1: Pre-fetch all sender details in one go
		senderIDs := make([]int64, 0, len(plan.Assignments))
		for id := range plan.Assignments {
			senderIDs = append(senderIDs, id)
		}
		senderDetails, err := s.senderRepo.FindAccountSenderDetailsByIDs(senderIDs)
		if err != nil {
			log.Printf("CRITICAL: Failed to pre-fetch sender details for page %d of task %d: %v. Skipping page.", page, task.ID, err)
			continue
		}
		senderMap := make(map[int64]model.AccountSender, len(senderDetails))
		for _, sender := range senderDetails {
			senderMap[sender.ID] = sender
		}

		var subjectTpl, bodyTpl *template.Template
		if task.TemplateID != nil {
			templateModel, err := s.templateRepo.FindByID(*task.TemplateID)
			if err != nil {
				log.Printf("Failed to load template ID %d for task %d, aborting dispatch: %v", *task.TemplateID, task.ID, err)
				return // Abort processing this task if template is not found
			}
			subjectTpl, _ = template.New("subject").Parse(templateModel.Subject)
			bodyTpl, _ = template.New("body").Parse(templateModel.Body)

		} else {
			// Handle ad-hoc tasks without a pre-defined template
			subjectStr, bodyStr := "", ""
			if task.Subject != nil {
				subjectStr = *task.Subject
			}
			if task.Body != nil {
				bodyStr = *task.Body
			}
			subjectTpl, _ = template.New("subject").Parse(subjectStr)
			bodyTpl, _ = template.New("body").Parse(bodyStr)
		}

		var aliyunTagName string
		if task.AliyunTagName != nil {
			aliyunTagName = *task.AliyunTagName
		}

		// === Batch-oriented Dispatch Logic for the current page ===
		recipientOffset := 0
		for senderID, count := range plan.Assignments {
			accountSender, ok := senderMap[senderID]
			if !ok {
				log.Printf("CRITICAL: Details for sender %d not found for page %d of task %d. Skipping %d emails.", senderID, page, task.ID, count)
				recipientOffset += count // Skip the recipients for this failed sender
				continue
			}

			// We need to use the recipients for the current page, starting from the offset
			pageRecipients := recipients
			if recipientOffset+count > len(pageRecipients) {
				log.Printf("Warning: Not enough recipients on page %d for sender %d assignment. Have %d, need %d.", page, senderID, len(pageRecipients)-recipientOffset, count)
				count = len(pageRecipients) - recipientOffset
			}

			recordsToCreate := make([]*model.EmailSendRecord, 0, count)
			for i := 0; i < count; i++ {
				recipient := pageRecipients[recipientOffset+i]

				renderedSubject, renderedBody, err := s.renderTemplateForRecipient(subjectTpl, bodyTpl, recipient)
				if err != nil {
					log.Printf("Error rendering template for recipient %s (Task %d, Page %d): %v", recipient.Email, task.ID, page, err)
					continue // Skip this recipient
				}

				record := &model.EmailSendRecord{
					TaskID:          &task.ID,
					AccountSenderID: accountSender.ID,
					RecipientEmail:  recipient.Email,
					Subject:         renderedSubject,
					Body:            renderedBody,
					Status:          model.RecordStatusPending,
				}
				recordsToCreate = append(recordsToCreate, record)
			}

			if len(recordsToCreate) == 0 {
				recipientOffset += count
				continue
			}

			if err := s.recordRepo.CreateBatch(recordsToCreate); err != nil {
				log.Printf("CRITICAL: Failed to batch create %d records for sender %d (Task %d, Page %d): %v", len(recordsToCreate), accountSender.ID, task.ID, page, err)
				recipientOffset += count
				continue
			}

			log.Printf("Successfully created %d records in batch for sender %d (Task %d, Page %d). Now enqueueing jobs.", len(recordsToCreate), accountSender.ID, task.ID, page)

			const chunkSize = 100
			for i := 0; i < len(recordsToCreate); i += chunkSize {
				end := i + chunkSize
				if end > len(recordsToCreate) {
					end = len(recordsToCreate)
				}
				chunk := recordsToCreate[i:end]

				emailInfos := make([]model.EmailInfo, 0, len(chunk))
				var commonSubject, commonBody string
				for j, record := range chunk {
					emailInfos = append(emailInfos, model.EmailInfo{
						RecordID:       record.ID,
						RecipientEmail: record.RecipientEmail,
					})
					if j == 0 {
						commonSubject = record.Subject
						commonBody = record.Body
					}
				}

				payload := model.EmailJobPayload{
					Emails:        emailInfos,
					Subject:       commonSubject,
					Body:          commonBody,
					AccountSender: accountSender,
					AliyunTagName: aliyunTagName,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					log.Printf("CRITICAL: Failed to marshal chunked job payload for task %d (Page %d): %v.", task.ID, page, err)
					continue
				}

				if err := s.queue.Enqueue(ctx, queue.EmailSendingQueue, string(payloadBytes)); err != nil {
					log.Printf("CRITICAL: Failed to enqueue chunked job for task %d (Page %d): %v.", task.ID, page, err)
				}
			}
			recipientOffset += count
		}
	}
}

// renderTemplateForRecipient renders the subject and body for a given recipient.
func (s *TaskDispatcherService) renderTemplateForRecipient(subjectTpl, bodyTpl *template.Template, recipient *model.Recipient) (string, string, error) {
	// --- FEATURE ENHANCEMENT: Combine recipient fields and metadata for templating ---
	var data map[string]interface{}
	if len(recipient.Metadata) > 0 {
		if err := json.Unmarshal(recipient.Metadata, &data); err != nil {
			return "", "", fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	} else {
		data = make(map[string]interface{})
	}

	// Add top-level recipient fields, allowing metadata to be overridden if keys conflict.
	data["ID"] = recipient.ID
	data["Email"] = recipient.Email
	data["FirstName"] = recipient.FirstName
	data["LastName"] = recipient.LastName
	// --- END FEATURE ENHANCEMENT ---

	var subjectBuf, bodyBuf bytes.Buffer

	// Render Subject
	if err := subjectTpl.Execute(&subjectBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	// Render Body
	if err := bodyTpl.Execute(&bodyBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to execute body template: %w", err)
	}

	return subjectBuf.String(), bodyBuf.String(), nil
}
