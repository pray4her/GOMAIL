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
	"email-service/pkg/aliyun"
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
	counterService  TaskCounterService
	queue           queue.QueueService
	pollingInterval time.Duration
	aliyunEndpoint  string
}

// NewTaskDispatcherService creates a new TaskDispatcherService.
func NewTaskDispatcherService(
	taskRepo repository.EmailTaskRepository,
	recordRepo repository.EmailSendRecordRepository,
	senderRepo repository.SenderRepository,
	templateRepo repository.TemplateRepository,
	groupService RecipientGroupService,
	loadBalancer LoadBalancerService,
	counterService TaskCounterService,
	queueSvc queue.QueueService,
	pollingInterval time.Duration,
	aliyunEndpoint string,
) *TaskDispatcherService {
	return &TaskDispatcherService{
		taskRepo:        taskRepo,
		recordRepo:      recordRepo,
		senderRepo:      senderRepo,
		templateRepo:    templateRepo,
		groupService:    groupService,
		loadBalancer:    loadBalancer,
		counterService:  counterService,
		queue:           queueSvc,
		pollingInterval: pollingInterval,
		aliyunEndpoint:  aliyunEndpoint,
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

	// Update task status to dispatching
	if err := s.taskRepo.UpdateStatus(task.ID, model.TaskStatusDispatching); err != nil {
		log.Printf("Error updating task %d status to dispatching: %v", task.ID, err)
		return
	}
	log.Printf("Task %d status updated to dispatching", task.ID)

	// === Optimization: Fetch constant data once before the loop ===
	group, err := s.groupService.GetGroup(*task.RecipientGroupID)
	if err != nil {
		log.Printf("CRITICAL: Failed to get recipient group %d for task %d: %v. Aborting dispatch.", *task.RecipientGroupID, task.ID, err)
		_ = s.taskRepo.UpdateStatus(task.ID, model.TaskStatusFailed)
		return
	}

	var subjectTpl, bodyTpl *template.Template
	if task.TemplateID != nil {
		templateModel, err := s.templateRepo.FindByID(*task.TemplateID)
		if err != nil {
			log.Printf("CRITICAL: Failed to load template ID %d for task %d, aborting dispatch: %v", *task.TemplateID, task.ID, err)
			_ = s.taskRepo.UpdateStatus(task.ID, model.TaskStatusFailed)
			return
		}
		subjectTpl, _ = template.New("subject").Parse(templateModel.Subject)
		bodyTpl, _ = template.New("body").Parse(templateModel.Body)
	} else {
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
	// === End Optimization ===

	// === Initialize task counter and update status to processing ===
	totalRecipientsInGroup, err := s.groupService.CountRecipients(group.ID, task.SendLimit, task.SendOffset)
	if err != nil {
		log.Printf("CRITICAL: Failed to count recipients for group %d (Task %d): %v. Aborting dispatch.", group.ID, task.ID, err)
		_ = s.taskRepo.UpdateStatus(task.ID, model.TaskStatusFailed)
		return
	}

	// The actual number of recipients for this task run is the total in the group,
	// adjusted by the user-defined limit.
	recipientsForThisRun := totalRecipientsInGroup
	if task.SendLimit != nil && *task.SendLimit < totalRecipientsInGroup {
		recipientsForThisRun = *task.SendLimit
	}

	if recipientsForThisRun == 0 {
		log.Printf("Task %d has no recipients to process (after applying limit/offset). Completing immediately.", task.ID)
		_ = s.taskRepo.UpdateStatus(task.ID, model.TaskStatusCompleted)
		_ = s.taskRepo.UpdateProgress(task.ID, 0, 0)
		return
	}

	// Initialize Redis counter with the actual number for this run
	if err := s.counterService.InitializeTaskCounter(task.ID, recipientsForThisRun); err != nil {
		log.Printf("CRITICAL: Failed to initialize counter for task %d: %v. Aborting dispatch.", task.ID, err)
		_ = s.taskRepo.UpdateStatus(task.ID, model.TaskStatusFailed)
		return
	}

	// Update task status to processing and set total recipients
	if err := s.taskRepo.UpdateStatus(task.ID, model.TaskStatusProcessing); err != nil {
		log.Printf("Error updating task %d status to processing: %v", task.ID, err)
		_ = s.counterService.DeleteTaskCounter(task.ID)
		return
	}
	if err := s.taskRepo.UpdateProgress(task.ID, 0, 0); err != nil {
		log.Printf("Error updating task %d progress: %v", task.ID, err)
	}

	// Update the task model with total recipients for database record
	task.TotalRecipients = recipientsForThisRun
	if err := s.taskRepo.Update(task); err != nil {
		log.Printf("Warning: Failed to update task %d total recipients in database: %v", task.ID, err)
	}

	log.Printf("Task %d initialized with %d total recipients, status updated to processing", task.ID, recipientsForThisRun)
	// === End Counter Initialization ===

	// === STREAMING REFACTOR: Process recipients in pages using search_after ===
	const pageSize = 10000 // Process 10,000 recipients per batch
	var searchAfter []interface{}
	var pageCount = 0
	recipientsToProcess := recipientsForThisRun // New: Track remaining recipients

	for recipientsToProcess > 0 {
		pageCount++
		log.Printf("Processing page %d for task %d...", pageCount, task.ID)

		// Determine the page size for the current request
		currentPageSize := pageSize
		if recipientsToProcess < pageSize {
			currentPageSize = recipientsToProcess
		}

		// Correctly resolve recipients for the current page without incorrect limit/offset
		recipients, nextSearchAfter, err := s.groupService.ResolveRecipients(group.ID, searchAfter, currentPageSize, task.SendLimit, task.SendOffset)
		if err != nil {
			log.Printf("CRITICAL: Failed to resolve recipients for page %d for group %d (Task %d): %v. Aborting task.", pageCount, group.ID, task.ID, err)
			return // Abort on recipient resolution failure
		}

		if len(recipients) == 0 {
			log.Printf("Finished processing all pages for task %d.", task.ID)
			break // This was the last page
		}
		// Update searchAfter for the next iteration
		searchAfter = nextSearchAfter

		recipientsToProcess -= len(recipients) // Decrement the counter

		log.Printf("Page %d for task %d resolved to %d recipients. Generating dispatch plan...", pageCount, task.ID, len(recipients))

		// === Load Balancing Logic for the current page ===
		plan, err := s.loadBalancer.GenerateDispatchPlan(task.CreatedByUserID, len(recipients))
		if err != nil {
			log.Printf("CRITICAL: Failed to generate dispatch plan for page %d of task %d: %v. Skipping page.", pageCount, task.ID, err)
			continue // Skip to next page
		}

		if !plan.Possible || plan.TotalEmails == 0 {
			log.Printf("Could not dispatch page %d of task %d. Requested %d emails, but plan allows for %d. Insufficient quota or no active senders.", pageCount, task.ID, len(recipients), plan.TotalEmails)
			continue // Skip to next page
		}
		log.Printf("Dispatch plan for page %d of task %d generated: %d emails to be sent using %d senders.", pageCount, task.ID, plan.TotalEmails, len(plan.Assignments))

		// Step 1: Pre-fetch all sender details in one go
		senderIDs := make([]int64, 0, len(plan.Assignments))
		for id := range plan.Assignments {
			senderIDs = append(senderIDs, id)
		}
		senderDetails, err := s.senderRepo.FindAccountSenderDetailsByIDs(senderIDs)
		if err != nil {
			log.Printf("CRITICAL: Failed to pre-fetch sender details for page %d of task %d: %v. Skipping page.", pageCount, task.ID, err)
			continue
		}
		senderMap := make(map[int64]model.AccountSender, len(senderDetails))
		for _, sender := range senderDetails {
			senderMap[sender.ID] = sender
		}

		// === Aliyun Tag Creation (first page only) ===
		if pageCount == 1 && (task.AliyunTagName == nil || *task.AliyunTagName == "") {
			tagName := fmt.Sprintf("task_%d_%d", task.ID, time.Now().Unix())
			uniqueAccounts := make(map[int64]model.Account)
			for _, sender := range senderDetails {
				if _, exists := uniqueAccounts[sender.Account.ID]; !exists {
					uniqueAccounts[sender.Account.ID] = sender.Account
				}
			}

			log.Printf("Task %d: Found %d unique Aliyun accounts for tag creation.", task.ID, len(uniqueAccounts))

			var tagCreationSuccess bool
			for _, account := range uniqueAccounts {
				aliyunClient, err := aliyun.NewClient(s.aliyunEndpoint, account.AccessKeyID, account.AccessKeySecret)
				if err != nil {
					log.Printf("Warning: Failed to create Aliyun client for account %d on task %d: %v", account.ID, task.ID, err)
					continue // Skip to next account
				}

				aliyunSender := aliyun.NewEmailSender(aliyunClient)
				tagDesc := fmt.Sprintf("Task: %s", task.TaskName)
				_, err = aliyunSender.CreateTag(tagName, tagDesc)
				if err != nil {
					log.Printf("Warning: Failed to create Aliyun tag '%s' for account %d on task %d: %v", tagName, account.ID, task.ID, err)
					// We continue even if one account fails, but we won't set the tag on the task unless at least one succeeds.
				} else {
					log.Printf("Successfully created Aliyun tag '%s' for account %d on task %d", tagName, account.ID, task.ID)
					tagCreationSuccess = true
				}
			}

			if tagCreationSuccess {
				task.AliyunTagName = &tagName
				if err := s.taskRepo.Update(task); err != nil {
					log.Printf("Warning: Failed to update task %d with Aliyun tag name '%s': %v", task.ID, tagName, err)
				} else {
					aliyunTagName = tagName // Ensure it's used in this same dispatch run
					log.Printf("Successfully associated Aliyun tag name '%s' with task %d", tagName, task.ID)
				}
			}
		}
		// === End Tag Creation ===

		// === Batch-oriented Dispatch Logic for the current page ===
		recipientOffset := 0
		for senderID, count := range plan.Assignments {
			accountSender, ok := senderMap[senderID]
			if !ok {
				log.Printf("CRITICAL: Details for sender %d not found for page %d of task %d. Skipping %d emails.", senderID, pageCount, task.ID, count)
				recipientOffset += count // Skip the recipients for this failed sender
				continue
			}

			// We need to use the recipients for the current page, starting from the offset
			pageRecipients := recipients
			if recipientOffset+count > len(pageRecipients) {
				log.Printf("Warning: Not enough recipients on page %d for sender %d assignment. Have %d, need %d.", pageCount, senderID, len(pageRecipients)-recipientOffset, count)
				count = len(pageRecipients) - recipientOffset
			}

			recordsToCreate := make([]*model.EmailSendRecord, 0, count)
			for i := 0; i < count; i++ {
				recipient := pageRecipients[recipientOffset+i]

				record := &model.EmailSendRecord{
					TaskID:          &task.ID,
					AccountSenderID: accountSender.ID,
					RecipientEmail:  recipient.Email,
					Status:          model.RecordStatusPending,
				}
				recordsToCreate = append(recordsToCreate, record)
			}

			if len(recordsToCreate) == 0 {
				recipientOffset += count
				continue
			}

			if err := s.recordRepo.CreateBatch(recordsToCreate); err != nil {
				log.Printf("CRITICAL: Failed to batch create %d records for sender %d (Task %d, Page %d): %v", len(recordsToCreate), accountSender.ID, task.ID, pageCount, err)
				recipientOffset += count
				continue
			}

			log.Printf("Successfully created %d records in batch for sender %d (Task %d, Page %d). Now enqueueing jobs.", len(recordsToCreate), accountSender.ID, task.ID, pageCount)

			// Enqueue a separate job for each created record.
			for _, record := range recordsToCreate {
				// We need to re-render the template here to get the body and subject for the job payload.
				// This is a trade-off: we re-render to avoid storing large bodies in the records table.
				recipient, _ := s.findRecipientByEmail(pageRecipients, record.RecipientEmail)
				renderedSubject, renderedBody, _ := s.renderTemplateForRecipient(subjectTpl, bodyTpl, recipient, &accountSender)

				payload := model.EmailJobPayload{
					RecordID:       record.ID,
					TaskID:         task.ID,
					RecipientEmail: record.RecipientEmail,
					Subject:        renderedSubject,
					Body:           renderedBody,
					AccountSender:  accountSender,
					AliyunTagName:  aliyunTagName,
				}

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					log.Printf("CRITICAL: Failed to marshal job payload for record %d: %v. Updating record to failed.", record.ID, err)
					errMsg := err.Error()
					_ = s.recordRepo.UpdateStatus(record.ID, model.RecordStatusFailed, nil, &errMsg)
					continue
				}

				if err := s.queue.Enqueue(ctx, queue.EmailSendingQueue, string(payloadBytes)); err != nil {
					log.Printf("CRITICAL: Failed to enqueue job for record %d: %v. Updating record to failed.", record.ID, err)
					errMsg := err.Error()
					_ = s.recordRepo.UpdateStatus(record.ID, model.RecordStatusFailed, nil, &errMsg)
				}
			}

			recipientOffset += count
		}
	}
}

func (s *TaskDispatcherService) findRecipientByEmail(recipients []*model.Recipient, email string) (*model.Recipient, bool) {
	for _, r := range recipients {
		if r.Email == email {
			return r, true
		}
	}
	return nil, false
}

// renderTemplateForRecipient renders the subject and body for a given recipient.
func (s *TaskDispatcherService) renderTemplateForRecipient(subjectTpl, bodyTpl *template.Template, recipient *model.Recipient, sender *model.AccountSender) (string, string, error) {
	// --- FEATURE ENHANCEMENT: Combine recipient and sender data for templating ---
	var recipientData map[string]interface{}
	if len(recipient.Metadata) > 0 {
		if err := json.Unmarshal(recipient.Metadata, &recipientData); err != nil {
			return "", "", fmt.Errorf("failed to unmarshal recipient metadata: %w", err)
		}
	} else {
		recipientData = make(map[string]interface{})
	}

	// Add top-level recipient fields, allowing metadata to be overridden if keys conflict.
	recipientData["ID"] = recipient.ID
	recipientData["Email"] = recipient.Email
	recipientData["FirstName"] = recipient.FirstName
	recipientData["LastName"] = recipient.LastName

	var senderData map[string]interface{}
	if sender != nil && len(sender.Metadata) > 0 {
		if err := json.Unmarshal(sender.Metadata, &senderData); err != nil {
			return "", "", fmt.Errorf("failed to unmarshal sender metadata: %w", err)
		}
	} else {
		senderData = make(map[string]interface{})
	}

	// Combine into a final data map for the template
	finalData := recipientData
	finalData["sender"] = senderData // Nest sender data under "Sender" key

	var subjectBuf, bodyBuf bytes.Buffer

	// Render Subject
	if err := subjectTpl.Execute(&subjectBuf, finalData); err != nil {
		return "", "", fmt.Errorf("failed to execute subject template: %w", err)
	}

	// Render Body
	if err := bodyTpl.Execute(&bodyBuf, finalData); err != nil {
		return "", "", fmt.Errorf("failed to execute body template: %w", err)
	}

	return subjectBuf.String(), bodyBuf.String(), nil
}
