package repository

import (
	"email-service/internal/model"
	"encoding/json"
	"fmt"
	"strings"

	"context"

	"github.com/elastic/go-elasticsearch/v8"
	"gorm.io/gorm"
)

// RecipientGroupRepository defines the interface for interacting with recipient group data.
type RecipientGroupRepository interface {
	Create(group *model.RecipientGroup) error
	FindByID(id int64) (*model.RecipientGroup, error)
	FindByName(name string) (*model.RecipientGroup, error)
	Update(group *model.RecipientGroup) error
	Delete(id int64) error
	List(page, pageSize int) ([]model.RecipientGroup, int64, error)

	// Methods for managing group members
	AddMembers(groupID int64, recipientIDs []int64) error
	RemoveMembers(groupID int64, recipientIDs []int64) error

	// Methods for resolving dynamic groups
	FindRecipientsByRules(rules []model.RecipientGroupRule, searchAfter []interface{}, pageSize int, limit *int, offset *int) ([]*model.Recipient, []interface{}, error)
	CountByRules(rules []model.RecipientGroupRule, limit *int, offset *int) (int64, error)
}

type recipientGroupRepository struct {
	db *gorm.DB
	es *elasticsearch.Client
}

// NewRecipientGroupRepository creates a new instance of RecipientGroupRepository.
func NewRecipientGroupRepository(db *gorm.DB, es *elasticsearch.Client) RecipientGroupRepository {
	return &recipientGroupRepository{db: db, es: es}
}

// Create creates a new recipient group, including its rules if it's dynamic.
func (r *recipientGroupRepository) Create(group *model.RecipientGroup) error {
	return r.db.Create(group).Error
}

// FindByID finds a recipient group by its ID, preloading its rules and a sample of members.
func (r *recipientGroupRepository) FindByID(id int64) (*model.RecipientGroup, error) {
	var group model.RecipientGroup
	err := r.db.Preload("Rules").Preload("CreatedByUser").First(&group, id).Error
	return &group, err
}

// FindByName finds a recipient group by its name.
func (r *recipientGroupRepository) FindByName(name string) (*model.RecipientGroup, error) {
	var group model.RecipientGroup
	err := r.db.Where("name = ?", name).First(&group).Error
	return &group, err
}

// Update updates an existing group's details and its rules.
func (r *recipientGroupRepository) Update(group *model.RecipientGroup) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(group).Error
}

// Delete deletes a recipient group by its ID.
func (r *recipientGroupRepository) Delete(id int64) error {
	return r.db.Delete(&model.RecipientGroup{}, id).Error
}

// List retrieves a paginated list of recipient groups.
func (r *recipientGroupRepository) List(page, pageSize int) ([]model.RecipientGroup, int64, error) {
	var groups []model.RecipientGroup
	var total int64

	r.db.Model(&model.RecipientGroup{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Preload("CreatedByUser").Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&groups).Error

	return groups, total, err
}

// AddMembers adds recipients to a static group.
func (r *recipientGroupRepository) AddMembers(groupID int64, recipientIDs []int64) error {
	if len(recipientIDs) == 0 {
		return nil
	}
	var members []model.RecipientGroupMember
	for _, rid := range recipientIDs {
		members = append(members, model.RecipientGroupMember{GroupID: groupID, RecipientID: rid})
	}
	return r.db.Create(&members).Error
}

// RemoveMembers removes recipients from a static group.
func (r *recipientGroupRepository) RemoveMembers(groupID int64, recipientIDs []int64) error {
	if len(recipientIDs) == 0 {
		return nil
	}
	return r.db.Where("group_id = ? AND recipient_id IN ?", groupID, recipientIDs).Delete(&model.RecipientGroupMember{}).Error
}

// buildESQuery builds the JSON query for Elasticsearch based on rules.
func (r *recipientGroupRepository) buildESQuery(rules []model.RecipientGroupRule) (string, error) {
	if len(rules) == 0 {
		return "", nil
	}

	var mustClauses []string
	for _, rule := range rules {
		field := rule.Field
		value, err := json.Marshal(rule.Value)
		if err != nil {
			return "", fmt.Errorf("failed to marshal rule value: %w", err)
		}

		var clause string
		switch strings.ToLower(rule.Operator) {
		case "eq", "equals":
			clause = fmt.Sprintf(`{"match": {"%s": %s}}`, field, string(value))
		case "neq", "not_equals":
			clause = fmt.Sprintf(`{"bool": {"must_not": {"match": {"%s": %s}}}}`, field, string(value))
		case "contains":
			clause = fmt.Sprintf(`{"match": {"%s": %s}}`, field, string(value))
		case "gt", "greater_than":
			clause = fmt.Sprintf(`{"range": {"%s": {"gt": %s}}}`, field, string(value))
		case "gte", "greater_than_or_equal":
			clause = fmt.Sprintf(`{"range": {"%s": {"gte": %s}}}`, field, string(value))
		case "lt", "less_than":
			clause = fmt.Sprintf(`{"range": {"%s": {"lt": %s}}}`, field, string(value))
		case "lte", "less_than_or_equal":
			clause = fmt.Sprintf(`{"range": {"%s": {"lte": %s}}}`, field, string(value))
		default:
			return "", fmt.Errorf("unsupported operator: %s", rule.Operator)
		}
		mustClauses = append(mustClauses, clause)
	}

	return fmt.Sprintf(`{ "bool": { "must": [ %s ] } }`, strings.Join(mustClauses, ",")), nil
}

// CountByRules directly asks Elasticsearch for the count of documents matching the rules.
func (r *recipientGroupRepository) CountByRules(rules []model.RecipientGroupRule, limit *int, offset *int) (int64, error) {
	if len(rules) == 0 {
		return 0, nil
	}

	queryPart, err := r.buildESQuery(rules)
	if err != nil {
		return 0, err
	}
	// === FIX: Build the full query with from and size for accurate counting ===
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`{`)
	queryBuilder.WriteString(fmt.Sprintf(`"query": %s,`, queryPart))
	// We don't need the source, just the count, but we need a search query.
	queryBuilder.WriteString(`"_source": false,`)
	// We use `track_total_hits: true` to get the full count regardless of pagination.
	// This is the number we need for the total count.
	queryBuilder.WriteString(`"track_total_hits": true`)
	queryBuilder.WriteString(`}`)

	query := queryBuilder.String()

	// Use the Search API instead of Count API to handle limit/offset logic implicitly later
	res, err := r.es.Search(
		r.es.Search.WithContext(context.Background()),
		r.es.Search.WithIndex("recipients"),
		r.es.Search.WithBody(strings.NewReader(query)),
		r.es.Search.WithSize(0), // We don't need documents, just the total count
	)

	if err != nil {
		return 0, fmt.Errorf("elasticsearch search for count failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("elasticsearch search for count returned an error: %s", res.String())
	}

	var esResponse struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return 0, fmt.Errorf("failed to decode elasticsearch count response: %w", err)
	}

	totalCount := esResponse.Hits.Total.Value

	// Apply offset logic
	start := 0
	if offset != nil {
		start = *offset
	}

	if int64(start) >= totalCount {
		return 0, nil // Offset is beyond the total number of documents
	}

	// Apply limit logic
	countAfterOffset := totalCount - int64(start)
	if limit != nil && int64(*limit) < countAfterOffset {
		return int64(*limit), nil
	}

	return countAfterOffset, nil
}

// FindRecipientsByRules dynamically builds and executes a query against Elasticsearch using the search_after parameter for deep pagination.
func (r *recipientGroupRepository) FindRecipientsByRules(rules []model.RecipientGroupRule, searchAfter []interface{}, pageSize int, limit *int, offset *int) ([]*model.Recipient, []interface{}, error) {
	if pageSize <= 0 {
		pageSize = 1000 // Default page size
	}

	queryPart, err := r.buildESQuery(rules)
	if err != nil {
		return nil, nil, err
	}

	// === FIX: Handle offset using search_after instead of from to avoid deep pagination limits ===
	// If we have an offset but no searchAfter, we need to build the searchAfter by skipping records
	if offset != nil && *offset > 0 && len(searchAfter) == 0 {
		// First, we need to get the search_after value for the offset position
		currentSearchAfter, err := r.getSearchAfterForOffset(rules, *offset)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build search_after for offset %d: %w", *offset, err)
		}
		searchAfter = currentSearchAfter
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`{`)
	queryBuilder.WriteString(fmt.Sprintf(`"query": %s,`, queryPart))
	queryBuilder.WriteString(`"_source": ["recipient_id"],`)

	// Apply limit and offset
	finalSize := pageSize
	if limit != nil {
		// If a limit is provided, it should be the maximum size we fetch.
		if *limit < finalSize {
			finalSize = *limit
		}
	}
	queryBuilder.WriteString(fmt.Sprintf(`"size": %d,`, finalSize))

	// No longer using "from" for deep pagination - always use search_after when available
	// Use recipient_id for a stable and unique sort order.
	queryBuilder.WriteString(`"sort": [{"recipient_id": "asc"}]`)

	if len(searchAfter) > 0 {
		searchAfterJSON, err := json.Marshal(searchAfter)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal search_after values: %w", err)
		}
		queryBuilder.WriteString(fmt.Sprintf(`, "search_after": %s`, string(searchAfterJSON)))
	}

	queryBuilder.WriteString(`}`)
	query := queryBuilder.String()

	// Execute the search request
	res, err := r.es.Search(
		r.es.Search.WithContext(context.Background()),
		r.es.Search.WithIndex("recipients"),
		r.es.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, nil, fmt.Errorf("elasticsearch search returned an error: %s", res.String())
	}

	// Decode the response
	var esResponse struct {
		Hits struct {
			Hits []struct {
				Source struct {
					RecipientID int64 `json:"recipient_id"`
				} `json:"_source"`
				Sort []interface{} `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, nil, fmt.Errorf("failed to decode elasticsearch response: %w", err)
	}

	// Extract IDs and fetch full recipient data from PostgreSQL
	if len(esResponse.Hits.Hits) == 0 {
		return []*model.Recipient{}, nil, nil
	}

	var recipientIDs []int64
	for _, hit := range esResponse.Hits.Hits {
		recipientIDs = append(recipientIDs, hit.Source.RecipientID)
	}

	var recipients []*model.Recipient
	if err := r.db.Where("id IN ?", recipientIDs).Find(&recipients).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to fetch recipients from db after es search: %w", err)
	}

	// Get the sort value from the last document to use for the next page
	nextSearchAfter := esResponse.Hits.Hits[len(esResponse.Hits.Hits)-1].Sort

	return recipients, nextSearchAfter, nil
}

// getSearchAfterForOffset builds a search_after value by skipping the specified number of records
func (r *recipientGroupRepository) getSearchAfterForOffset(rules []model.RecipientGroupRule, offset int) ([]interface{}, error) {
	const maxSingleQuery = 10000 // ES limit for from+size

	queryPart, err := r.buildESQuery(rules)
	if err != nil {
		return nil, err
	}

	var currentSearchAfter []interface{}
	remainingOffset := offset

	for remainingOffset > 0 {
		// Calculate the size for this iteration
		querySize := maxSingleQuery
		if remainingOffset < maxSingleQuery {
			querySize = remainingOffset
		}

		var queryBuilder strings.Builder
		queryBuilder.WriteString(`{`)
		queryBuilder.WriteString(fmt.Sprintf(`"query": %s,`, queryPart))
		queryBuilder.WriteString(`"_source": ["recipient_id"],`)
		queryBuilder.WriteString(fmt.Sprintf(`"size": %d,`, querySize))
		queryBuilder.WriteString(`"sort": [{"recipient_id": "asc"}]`)

		if len(currentSearchAfter) > 0 {
			searchAfterJSON, err := json.Marshal(currentSearchAfter)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal search_after in offset building: %w", err)
			}
			queryBuilder.WriteString(fmt.Sprintf(`, "search_after": %s`, string(searchAfterJSON)))
		}

		queryBuilder.WriteString(`}`)

		// Execute the search request
		res, err := r.es.Search(
			r.es.Search.WithContext(context.Background()),
			r.es.Search.WithIndex("recipients"),
			r.es.Search.WithBody(strings.NewReader(queryBuilder.String())),
		)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch search failed during offset building: %w", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return nil, fmt.Errorf("elasticsearch search returned an error during offset building: %s", res.String())
		}

		var esResponse struct {
			Hits struct {
				Hits []struct {
					Sort []interface{} `json:"sort"`
				} `json:"hits"`
			} `json:"hits"`
		}

		if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
			return nil, fmt.Errorf("failed to decode elasticsearch response during offset building: %w", err)
		}

		if len(esResponse.Hits.Hits) == 0 {
			// No more records available, return what we have
			break
		}

		// Update the search_after for next iteration
		currentSearchAfter = esResponse.Hits.Hits[len(esResponse.Hits.Hits)-1].Sort
		remainingOffset -= len(esResponse.Hits.Hits)
	}

	return currentSearchAfter, nil
}
