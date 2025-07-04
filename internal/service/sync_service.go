package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"email-service/internal/model"
	"email-service/internal/queue"
	"email-service/internal/repository"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/gin-gonic/gin"
)

const (
	esRecipientIndex = "recipients"
)

// SyncService is responsible for keeping Elasticsearch in sync with the primary database.
type SyncService struct {
	queue         queue.QueueService
	recipientRepo repository.RecipientRepository
	esClient      *elasticsearch.Client
}

// NewSyncService creates a new SyncService.
func NewSyncService(
	queueSvc queue.QueueService,
	recipientRepo repository.RecipientRepository,
	esClient *elasticsearch.Client,
) *SyncService {
	return &SyncService{
		queue:         queueSvc,
		recipientRepo: recipientRepo,
		esClient:      esClient,
	}
}

// Start launches the background worker for syncing recipients.
func (s *SyncService) Start(ctx context.Context) {
	log.Println("Starting Elasticsearch sync service...")
	go s.runRecipientSyncWorker(ctx)
}

func (s *SyncService) runRecipientSyncWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Recipient sync worker shutting down.")
			return
		default:
			// Dequeue a recipient ID that needs syncing
			recipientIDStr, err := s.queue.Dequeue(ctx, queue.RecipientSyncQueue)
			if err != nil {
				log.Printf("Error dequeuing recipient for sync: %v", err)
				continue
			}

			recipientID, err := strconv.ParseUint(recipientIDStr, 10, 64)
			if err != nil {
				log.Printf("Error parsing recipient ID '%s' for sync: %v", recipientIDStr, err)
				continue
			}

			log.Printf("Syncing recipient %d to Elasticsearch...", recipientID)
			if err := s.syncRecipient(ctx, uint(recipientID)); err != nil {
				log.Printf("Failed to sync recipient %d: %v", recipientID, err)
				// A robust system might re-queue the message with a backoff strategy.
			}
		}
	}
}

// syncRecipient fetches a recipient from the database and indexes it into Elasticsearch.
func (s *SyncService) syncRecipient(ctx context.Context, id uint) error {
	recipient, err := s.recipientRepo.FindByID(id)
	if err != nil {
		return err
	}

	// Prepare the document for Elasticsearch
	doc, err := s.prepareESDocument(recipient)
	if err != nil {
		return err
	}

	req := esapi.IndexRequest{
		Index:      esRecipientIndex,
		DocumentID: strconv.FormatUint(uint64(recipient.ID), 10),
		Body:       strings.NewReader(doc),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, s.esClient)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return_str := "Error indexing recipient " + string(rune(id)) + ": " + res.String()
		return errors.New(return_str)
	}

	log.Printf("Successfully indexed recipient %d", id)
	return nil
}

// prepareESDocument converts a recipient model to an Elasticsearch-compatible JSON string.
func (s *SyncService) prepareESDocument(recipient *model.Recipient) (string, error) {
	// Unmarshal metadata so we can include it at the top level in the ES doc
	var metadata map[string]interface{}
	if recipient.Metadata != nil {
		if err := json.Unmarshal(recipient.Metadata, &metadata); err != nil {
			log.Printf("Warning: Could not unmarshal metadata for recipient %d: %v", recipient.ID, err)
			metadata = make(map[string]interface{})
		}
	}

	esDoc := gin.H{
		"recipient_id": recipient.ID,
		"email":        recipient.Email,
		"first_name":   recipient.FirstName,
		"last_name":    recipient.LastName,
		"status":       recipient.Status,
		"created_at":   recipient.CreatedAt,
		"updated_at":   recipient.UpdatedAt,
		"metadata":     metadata,
	}

	docBytes, err := json.Marshal(esDoc)
	if err != nil {
		return "", err
	}
	return string(docBytes), nil
}
