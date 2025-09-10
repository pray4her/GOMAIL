package repository

import (
	"context"
	"email-service/internal/model"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"gorm.io/gorm"
)

// RecipientRepository defines the interface for interacting with recipients.
type RecipientRepository interface {
	Create(recipient *model.Recipient) error
	Update(recipient *model.Recipient) error
	FindByID(id uint) (*model.Recipient, error)
	FindByIds(ids []uint) ([]model.Recipient, error)
	FindByEmail(email string) (*model.Recipient, error)
	List(page, pageSize int, filters map[string]string) ([]model.Recipient, int64, error)
	ListWithSearchAfter(searchAfter []interface{}, pageSize int, filters map[string]string) ([]model.Recipient, []interface{}, int64, error)
	Delete(id uint) error
	FindByGroupID(groupID int64, page, pageSize int) ([]*model.Recipient, error)
	CountByGroupID(groupID int64) (int64, error)
	CreateBatch(recipients []*model.Recipient) ([]*model.Recipient, []error)
	BatchSyncToES(recipients []*model.Recipient) error
}

type recipientRepository struct {
	db *gorm.DB
	es *elasticsearch.Client
}

// NewRecipientRepository creates a new RecipientRepository.
func NewRecipientRepository(db *gorm.DB, es *elasticsearch.Client) RecipientRepository {
	return &recipientRepository{db: db, es: es}
}

func (r *recipientRepository) Create(recipient *model.Recipient) error {
	return r.db.Create(recipient).Error
}

func (r *recipientRepository) Update(recipient *model.Recipient) error {
	return r.db.Save(recipient).Error
}

func (r *recipientRepository) FindByID(id uint) (*model.Recipient, error) {
	var recipient model.Recipient
	err := r.db.First(&recipient, id).Error
	return &recipient, err
}

func (r *recipientRepository) FindByIds(ids []uint) ([]model.Recipient, error) {
	var recipients []model.Recipient
	err := r.db.Where("id IN ?", ids).Find(&recipients).Error
	return recipients, err
}

func (r *recipientRepository) FindByEmail(email string) (*model.Recipient, error) {
	var recipient model.Recipient
	err := r.db.Where("email = ?", email).First(&recipient).Error
	return &recipient, err
}

func (r *recipientRepository) List(page, pageSize int, filters map[string]string) ([]model.Recipient, int64, error) {
	// Check for deep pagination limit
	const maxFromSize = 10000
	if (page-1)*pageSize >= maxFromSize {
		return nil, 0, fmt.Errorf("deep pagination detected: from+size (%d) exceeds Elasticsearch limit (%d). Please use search_after pagination for deep pages", (page-1)*pageSize+pageSize, maxFromSize)
	}

	if r.es == nil {
		// Fallback to DB if ES is not configured (filtering won't work here)
		var recipients []model.Recipient
		var total int64
		offset := (page - 1) * pageSize
		if err := r.db.Model(&model.Recipient{}).Count(&total).Error; err != nil {
			return nil, 0, err
		}
		err := r.db.Limit(pageSize).Offset(offset).Order("created_at desc").Find(&recipients).Error
		return recipients, total, err
	}

	var recipients []model.Recipient
	var total int64

	// Build the Elasticsearch query
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`{`)

	// Build the query part based on filters
	var mustClauses []string
	if len(filters) == 0 {
		queryBuilder.WriteString(`"query": {"match_all": {}}`)
	} else {
		for key, value := range filters {
			var clause string
			if key == "email" {
				clause = fmt.Sprintf(`{"term": {"email": "%s"}}`, value)
			} else if key == "name" {
				// Use multi_match for fuzzy search on name fields
				clause = fmt.Sprintf(`{"multi_match": {"query": "%s", "fields": ["first_name", "last_name"], "fuzziness": "AUTO"}}`, value)
			} else if strings.HasPrefix(key, "metadata.") {
				// Existing metadata filtering
				metadataKey := strings.TrimPrefix(key, "metadata.")
				clause = fmt.Sprintf(`{"term": {"metadata.%s.keyword": "%s"}}`, metadataKey, value)
			}
			if clause != "" {
				mustClauses = append(mustClauses, clause)
			}
		}

		if len(mustClauses) > 0 {
			queryBuilder.WriteString(fmt.Sprintf(`"query": {"bool": {"must": [%s]}}`, strings.Join(mustClauses, ",")))
		} else {
			queryBuilder.WriteString(`"query": {"match_all": {}}`)
		}
	}

	queryBuilder.WriteString(fmt.Sprintf(`, "from": %d`, (page-1)*pageSize))
	queryBuilder.WriteString(fmt.Sprintf(`, "size": %d`, pageSize))

	// Sort by relevance score if a name search is performed, otherwise by creation date
	if _, hasNameFilter := filters["name"]; !hasNameFilter {
		queryBuilder.WriteString(`, "sort": [{"created_at": "desc"}]`)
	}

	queryBuilder.WriteString(`}`)

	res, err := r.es.Search(
		r.es.Search.WithContext(context.Background()),
		r.es.Search.WithIndex("recipients"),
		r.es.Search.WithBody(strings.NewReader(queryBuilder.String())),
		r.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("elasticsearch search returned an error: %s", res.String())
	}

	var esResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, 0, fmt.Errorf("failed to decode elasticsearch response: %w", err)
	}

	total = esResponse.Hits.Total.Value
	for _, hit := range esResponse.Hits.Hits {
		var recipient model.Recipient
		// The source from ES contains recipient_id, but our model has ID. We need to map it.
		var sourceMap map[string]interface{}
		if err := json.Unmarshal(hit.Source, &sourceMap); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal hit source: %w", err)
		}

		// Manual mapping to handle the ID field difference
		if id, ok := sourceMap["recipient_id"].(float64); ok { // JSON numbers are float64
			sourceMap["id"] = int64(id)
		}

		// Re-marshal to recipient struct
		remappedBytes, err := json.Marshal(sourceMap)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to re-marshal source map: %w", err)
		}

		if err := json.Unmarshal(remappedBytes, &recipient); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal remapped source to recipient: %w", err)
		}

		recipients = append(recipients, recipient)
	}

	return recipients, total, nil
}

// ListWithSearchAfter implements deep pagination using Elasticsearch search_after mechanism
func (r *recipientRepository) ListWithSearchAfter(searchAfter []interface{}, pageSize int, filters map[string]string) ([]model.Recipient, []interface{}, int64, error) {
	if r.es == nil {
		return nil, nil, 0, fmt.Errorf("elasticsearch client is not configured, cannot use search_after pagination")
	}

	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}

	var recipients []model.Recipient

	// Build the Elasticsearch query
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`{`)

	// Build the query part based on filters
	var mustClauses []string
	if len(filters) == 0 {
		queryBuilder.WriteString(`"query": {"match_all": {}}`)
	} else {
		for key, value := range filters {
			var clause string
			if key == "email" {
				clause = fmt.Sprintf(`{"term": {"email": "%s"}}`, value)
			} else if key == "name" {
				// Use multi_match for fuzzy search on name fields
				clause = fmt.Sprintf(`{"multi_match": {"query": "%s", "fields": ["first_name", "last_name"], "fuzziness": "AUTO"}}`, value)
			} else if strings.HasPrefix(key, "metadata.") {
				// Existing metadata filtering
				metadataKey := strings.TrimPrefix(key, "metadata.")
				clause = fmt.Sprintf(`{"term": {"metadata.%s.keyword": "%s"}}`, metadataKey, value)
			}
			if clause != "" {
				mustClauses = append(mustClauses, clause)
			}
		}

		if len(mustClauses) > 0 {
			queryBuilder.WriteString(fmt.Sprintf(`"query": {"bool": {"must": [%s]}}`, strings.Join(mustClauses, ",")))
		} else {
			queryBuilder.WriteString(`"query": {"match_all": {}}`)
		}
	}

	queryBuilder.WriteString(fmt.Sprintf(`, "size": %d`, pageSize))
	// Use recipient_id for stable and unique sort order
	queryBuilder.WriteString(`, "sort": [{"recipient_id": "asc"}]`)

	// Add search_after if provided
	if len(searchAfter) > 0 {
		searchAfterJSON, err := json.Marshal(searchAfter)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to marshal search_after values: %w", err)
		}
		queryBuilder.WriteString(fmt.Sprintf(`, "search_after": %s`, string(searchAfterJSON)))
	}

	queryBuilder.WriteString(`}`)

	// Execute the search request
	res, err := r.es.Search(
		r.es.Search.WithContext(context.Background()),
		r.es.Search.WithIndex("recipients"),
		r.es.Search.WithBody(strings.NewReader(queryBuilder.String())),
		r.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, nil, 0, fmt.Errorf("elasticsearch search returned an error: %s", res.String())
	}

	// Decode the response
	var esResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source json.RawMessage `json:"_source"`
				Sort   []interface{}   `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, nil, 0, fmt.Errorf("failed to decode elasticsearch response: %w", err)
	}

	total := esResponse.Hits.Total.Value
	var nextSearchAfter []interface{}

	for _, hit := range esResponse.Hits.Hits {
		var recipient model.Recipient
		// The source from ES contains recipient_id, but our model has ID. We need to map it.
		var sourceMap map[string]interface{}
		if err := json.Unmarshal(hit.Source, &sourceMap); err != nil {
			return nil, nil, 0, fmt.Errorf("failed to unmarshal hit source: %w", err)
		}

		// Manual mapping to handle the ID field difference
		if id, ok := sourceMap["recipient_id"].(float64); ok { // JSON numbers are float64
			sourceMap["id"] = int64(id)
		}

		// Re-marshal to recipient struct
		remappedBytes, err := json.Marshal(sourceMap)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to re-marshal source map: %w", err)
		}

		if err := json.Unmarshal(remappedBytes, &recipient); err != nil {
			return nil, nil, 0, fmt.Errorf("failed to unmarshal remapped source to recipient: %w", err)
		}

		recipients = append(recipients, recipient)

		// Store the last sort value for next pagination
		if len(hit.Sort) > 0 {
			nextSearchAfter = hit.Sort
		}
	}

	return recipients, nextSearchAfter, total, nil
}

func (r *recipientRepository) Delete(id uint) error {
	// Also delete from elasticsearch
	if r.es != nil {
		req := esapi.DeleteRequest{
			Index:      "recipients",
			DocumentID: strconv.FormatUint(uint64(id), 10),
			Refresh:    "true",
		}
		res, err := req.Do(context.Background(), r.es)

		if err != nil {
			// Log the error but don't fail the whole operation, DB is the source of truth
			fmt.Printf("warn: failed to delete recipient %d from elasticsearch: %v\n", id, err)
		} else {
			defer res.Body.Close()
			if res.IsError() && res.StatusCode != 404 { // 404 is not an error in this context
				fmt.Printf("warn: elasticsearch returned an error while deleting recipient %d: %s\n", id, res.String())
			}
		}
	}
	return r.db.Delete(&model.Recipient{}, id).Error
}

func (r *recipientRepository) FindByGroupID(groupID int64, page, pageSize int) ([]*model.Recipient, error) {
	var recipients []*model.Recipient
	offset := (page - 1) * pageSize
	err := r.db.
		Joins("JOIN recipient_group_members ON recipient_group_members.recipient_id = recipients.id").
		Where("recipient_group_members.group_id = ?", groupID).
		Limit(pageSize).
		Offset(offset).
		Order("recipients.id ASC").
		Find(&recipients).Error
	return recipients, err
}

func (r *recipientRepository) CountByGroupID(groupID int64) (int64, error) {
	var count int64
	err := r.db.
		Model(&model.Recipient{}).
		Joins("JOIN recipient_group_members ON recipient_group_members.recipient_id = recipients.id").
		Where("recipient_group_members.group_id = ?", groupID).
		Count(&count).Error
	return count, err
}

func (r *recipientRepository) CreateBatch(recipients []*model.Recipient) ([]*model.Recipient, []error) {
	var successRecipients []*model.Recipient
	var errors []error

	if len(recipients) == 0 {
		return successRecipients, errors
	}

	// Use transaction for batch insert
	tx := r.db.Begin()
	if tx.Error != nil {
		errors = append(errors, tx.Error)
		return successRecipients, errors
	}

	// Use CreateInBatches for better performance
	batchSize := 100
	if err := tx.CreateInBatches(recipients, batchSize).Error; err != nil {
		tx.Rollback()
		// If batch insert fails, try individual inserts to identify specific failures
		for _, recipient := range recipients {
			if err := r.db.Create(recipient).Error; err != nil {
				errors = append(errors, fmt.Errorf("failed to create recipient %s: %w", recipient.Email, err))
			} else {
				successRecipients = append(successRecipients, recipient)
			}
		}
	} else {
		// Batch insert succeeded
		tx.Commit()
		successRecipients = recipients
	}

	return successRecipients, errors
}

func (r *recipientRepository) BatchSyncToES(recipients []*model.Recipient) error {
	if r.es == nil || len(recipients) == 0 {
		return nil
	}

	var bulkBody strings.Builder
	for _, recipient := range recipients {
		// Create index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": "recipients",
				"_id":    strconv.FormatInt(recipient.ID, 10),
			},
		}
		actionBytes, _ := json.Marshal(action)
		bulkBody.Write(actionBytes)
		bulkBody.WriteString("\n")

		// Create document body
		doc := map[string]interface{}{
			"recipient_id": recipient.ID,
			"email":        recipient.Email,
			"first_name":   recipient.FirstName,
			"last_name":    recipient.LastName,
			"status":       recipient.Status,
			"created_at":   recipient.CreatedAt,
			"updated_at":   recipient.UpdatedAt,
		}

		// Handle metadata
		if len(recipient.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(recipient.Metadata, &metadata); err == nil {
				doc["metadata"] = metadata
			}
		}

		docBytes, _ := json.Marshal(doc)
		bulkBody.Write(docBytes)
		bulkBody.WriteString("\n")
	}

	// Execute bulk request
	res, err := r.es.Bulk(
		strings.NewReader(bulkBody.String()),
		r.es.Bulk.WithIndex("recipients"),
		r.es.Bulk.WithRefresh("true"),
	)
	if err != nil {
		return fmt.Errorf("bulk sync to elasticsearch failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch bulk sync returned an error: %s", res.String())
	}

	return nil
}
