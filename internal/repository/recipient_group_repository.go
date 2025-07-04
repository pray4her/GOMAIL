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
	FindRecipientsByRules(rules []model.RecipientGroupRule, page, pageSize int) ([]*model.Recipient, error)
	CountByRules(rules []model.RecipientGroupRule) (int64, error)
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
func (r *recipientGroupRepository) CountByRules(rules []model.RecipientGroupRule) (int64, error) {
	if len(rules) == 0 {
		return 0, nil
	}

	queryPart, err := r.buildESQuery(rules)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf(`{ "query": %s }`, queryPart)

	res, err := r.es.Count(
		r.es.Count.WithContext(context.Background()),
		r.es.Count.WithIndex("recipients"),
		r.es.Count.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		return 0, fmt.Errorf("elasticsearch count failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("elasticsearch count returned an error: %s", res.String())
	}

	var esResponse struct {
		Count int64 `json:"count"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return 0, fmt.Errorf("failed to decode elasticsearch count response: %w", err)
	}

	return esResponse.Count, nil
}

// FindRecipientsByRules dynamically builds and executes a query against Elasticsearch with pagination.
func (r *recipientGroupRepository) FindRecipientsByRules(rules []model.RecipientGroupRule, page, pageSize int) ([]*model.Recipient, error) {
	if len(rules) == 0 {
		return []*model.Recipient{}, nil
	}

	queryPart, err := r.buildESQuery(rules)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`{
		"query": %s,
		"_source": ["recipient_id"],
		"from": %d,
		"size": %d
	}`, queryPart, offset, pageSize)

	// Execute the search request
	res, err := r.es.Search(
		r.es.Search.WithContext(context.Background()),
		r.es.Search.WithIndex("recipients"),
		r.es.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch search returned an error: %s", res.String())
	}

	// Decode the response
	var esResponse struct {
		Hits struct {
			Hits []struct {
				Source struct {
					RecipientID int64 `json:"recipient_id"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode elasticsearch response: %w", err)
	}

	// Extract IDs and fetch full recipient data from PostgreSQL (Source of Truth)
	var recipientIDs []int64
	for _, hit := range esResponse.Hits.Hits {
		recipientIDs = append(recipientIDs, hit.Source.RecipientID)
	}

	if len(recipientIDs) == 0 {
		return []*model.Recipient{}, nil
	}

	var recipients []*model.Recipient
	if err := r.db.Where("id IN ?", recipientIDs).Find(&recipients).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch recipients from db after es search: %w", err)
	}

	return recipients, nil
}
