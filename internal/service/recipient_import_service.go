package service

import (
	"context"
	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// RecipientImportService defines the interface for recipient import operations.
type RecipientImportService interface {
	CreateImportTask(userID int64, taskName, fileName string, fileHeader *multipart.FileHeader) (*model.RecipientImportTask, error)
	GetImportTask(taskID int64) (*model.RecipientImportTask, error)
	ListImportTasks(userID int64, page, pageSize int) ([]*model.RecipientImportTask, int64, error)
	ProcessImportTask(taskID int64) error
}

type recipientImportService struct {
	importRepo     repository.RecipientImportTaskRepository
	recipientRepo  repository.RecipientRepository
	queueService   queue.QueueService
	fileProcessor  FileProcessor
	fileUploadPath string
}

// NewRecipientImportService creates a new RecipientImportService.
func NewRecipientImportService(
	importRepo repository.RecipientImportTaskRepository,
	recipientRepo repository.RecipientRepository,
	queueService queue.QueueService,
	fileProcessor FileProcessor,
	fileUploadPath string,
) RecipientImportService {
	return &recipientImportService{
		importRepo:     importRepo,
		recipientRepo:  recipientRepo,
		queueService:   queueService,
		fileProcessor:  fileProcessor,
		fileUploadPath: fileUploadPath,
	}
}

func (s *recipientImportService) CreateImportTask(userID int64, taskName, fileName string, fileHeader *multipart.FileHeader) (*model.RecipientImportTask, error) {
	// Validate file type
	fileExt := strings.ToLower(filepath.Ext(fileName))
	var fileType string
	switch fileExt {
	case ".csv":
		fileType = "csv"
	case ".xlsx", ".xls":
		fileType = "excel"
	case ".json":
		fileType = "json"
	default:
		return nil, fmt.Errorf("unsupported file type: %s. Supported types: csv, xlsx, json", fileExt)
	}

	// Ensure upload directory exists
	uploadPath, err := filepath.Abs(s.fileUploadPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute upload path: %w", err)
	}
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate unique filename and join with the absolute path
	timestamp := time.Now().Format("20060102_150405")
	uniqueFileName := fmt.Sprintf("%d_%s_%s", userID, timestamp, fileName)
	filePath := filepath.Join(uploadPath, uniqueFileName)

	// Save uploaded file
	if err := s.saveUploadedFile(fileHeader, filePath); err != nil {
		return nil, fmt.Errorf("failed to save uploaded file: %w", err)
	}

	// Create import task record
	task := &model.RecipientImportTask{
		TaskName:        taskName,
		FileName:        uniqueFileName,
		FileSize:        fileHeader.Size,
		FileType:        fileType,
		Status:          model.ImportTaskStatusPending,
		CreatedByUserID: userID,
	}

	if err := s.importRepo.Create(task); err != nil {
		// Clean up uploaded file if task creation fails
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create import task: %w", err)
	}

	// Enqueue task for processing
	if err := s.queueService.Enqueue(context.Background(), queue.RecipientImportQueue, strconv.FormatInt(task.ID, 10)); err != nil {
		return nil, fmt.Errorf("failed to enqueue import task: %w", err)
	}

	return task, nil
}

func (s *recipientImportService) GetImportTask(taskID int64) (*model.RecipientImportTask, error) {
	return s.importRepo.FindByID(taskID)
}

func (s *recipientImportService) ListImportTasks(userID int64, page, pageSize int) ([]*model.RecipientImportTask, int64, error) {
	return s.importRepo.FindByUserID(userID, page, pageSize)
}

func (s *recipientImportService) ProcessImportTask(taskID int64) error {
	// Get task details
	task, err := s.importRepo.FindByID(taskID)
	if err != nil {
		return fmt.Errorf("failed to find import task %d: %w", taskID, err)
	}

	if task.Status != model.ImportTaskStatusPending && task.Status != model.ImportTaskStatusProcessing {
		return fmt.Errorf("task %d is not in pending or processing status, current status: %s", taskID, task.Status)
	}

	// Check if this is a recovery operation
	isRecovery := task.Status == model.ImportTaskStatusProcessing
	startOffset := task.ProcessedRecords

	if isRecovery {
		log.Printf("Resuming import task %d from record %d", taskID, startOffset)
	} else {
		// Update task status to processing for new tasks
		if err := s.importRepo.UpdateStatus(taskID, model.ImportTaskStatusProcessing, nil); err != nil {
			return fmt.Errorf("failed to update task status to processing: %w", err)
		}

		// Update start time for new tasks
		if err := s.importRepo.UpdateStartTime(taskID, time.Now()); err != nil {
			return fmt.Errorf("failed to update task start time: %w", err)
		}
	}

	// Construct the full file path
	uploadPath, err := filepath.Abs(s.fileUploadPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute upload path for task %d: %w", taskID, err)
	}
	filePath := filepath.Join(uploadPath, task.FileName)

	// Process the file
	if err := s.processFile(task, filePath, startOffset); err != nil {
		// Update task with error
		errorMsg := err.Error()
		s.importRepo.UpdateStatus(taskID, model.ImportTaskStatusFailed, &errorMsg)
		return fmt.Errorf("failed to process file: %w", err)
	}

	// Only delete file after successful completion
	if err := os.Remove(filePath); err != nil {
		log.Printf("Warning: failed to delete completed import file %s: %v", filePath, err)
	} else {
		log.Printf("Successfully deleted completed import file: %s", filePath)
	}

	return nil
}

func (s *recipientImportService) processFile(task *model.RecipientImportTask, filePath string, startOffset int) error {
	// Parse file based on type
	var rawData []map[string]interface{}
	var err error

	switch task.FileType {
	case "csv":
		rawData, err = s.fileProcessor.ParseCSVWithOffset(filePath, startOffset)
	case "excel":
		rawData, err = s.fileProcessor.ParseExcelWithOffset(filePath, startOffset)
	case "json":
		rawData, err = s.fileProcessor.ParseJSONWithOffset(filePath, startOffset)
	default:
		return fmt.Errorf("unsupported file type: %s", task.FileType)
	}

	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	// If this is a recovery, get total records from task, otherwise calculate from parsed data
	var totalRecords int
	if startOffset > 0 {
		totalRecords = task.TotalRecords
	} else {
		// For new tasks, get total records from full file parsing first
		var fullData []map[string]interface{}
		switch task.FileType {
		case "csv":
			fullData, err = s.fileProcessor.ParseCSV(filePath)
		case "excel":
			fullData, err = s.fileProcessor.ParseExcel(filePath)
		case "json":
			fullData, err = s.fileProcessor.ParseJSON(filePath)
		}
		if err != nil {
			return fmt.Errorf("failed to get total records count: %w", err)
		}
		totalRecords = len(fullData)
		// Update total records count initially for new tasks
		if err := s.importRepo.UpdateProgress(task.ID, startOffset, task.SuccessRecords, task.FailedRecords, totalRecords); err != nil {
			return fmt.Errorf("failed to update initial progress: %w", err)
		}
	}

	// Process records in batches
	batchSize := 100
	var totalSuccess, totalFailed int

	// Initialize with existing counts for recovery
	totalSuccess = task.SuccessRecords
	totalFailed = task.FailedRecords

	for i := 0; i < len(rawData); i += batchSize {
		end := i + batchSize
		if end > len(rawData) {
			end = len(rawData)
		}

		batch := rawData[i:end]
		successCount, failureCount := s.processBatch(task, batch)

		totalSuccess += successCount
		totalFailed += failureCount

		// Update progress with actual processed position
		currentProcessed := startOffset + i + len(batch)
		if err := s.importRepo.UpdateProgress(task.ID, currentProcessed, totalSuccess, totalFailed, totalRecords); err != nil {
			return fmt.Errorf("failed to update progress: %w", err)
		}
	}

	// Final status update
	completedAt := time.Now()
	finalStatus := model.ImportTaskStatusCompleted
	if totalFailed > 0 && totalSuccess == 0 {
		finalStatus = model.ImportTaskStatusFailed
	}

	if err := s.importRepo.UpdateResult(task.ID, finalStatus, &completedAt); err != nil {
		return fmt.Errorf("failed to update final result: %w", err)
	}

	return nil
}

func (s *recipientImportService) processBatch(task *model.RecipientImportTask, batch []map[string]interface{}) (successCount int, failureCount int) {
	var validRecipients []*model.Recipient

	for _, rawData := range batch {
		recipient, err := s.fileProcessor.ValidateRecipientData(rawData)
		if err != nil {
			failureCount++
			continue
		}
		validRecipients = append(validRecipients, recipient)
	}

	if len(validRecipients) > 0 {
		successRecipients, errors := s.recipientRepo.CreateBatch(validRecipients)
		successCount = len(successRecipients)
		failureCount += len(errors)

		if len(successRecipients) > 0 {
			if err := s.recipientRepo.BatchSyncToES(successRecipients); err != nil {
				fmt.Printf("Warning: Failed to sync batch to Elasticsearch: %v\n", err)
			}
		}
	}

	return successCount, failureCount
}

func (s *recipientImportService) saveUploadedFile(fileHeader *multipart.FileHeader, destination string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func getEmailFromRawData(data map[string]interface{}) string {
	if email, ok := data["email"].(string); ok {
		return email
	}
	return "unknown"
}
