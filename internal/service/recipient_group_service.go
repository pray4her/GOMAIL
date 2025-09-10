package service

import (
	"email-service/internal/model"
	"email-service/internal/repository"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// RecipientGroupService defines the interface for recipient group business logic.
type RecipientGroupService interface {
	CreateGroup(group *model.RecipientGroup) (*model.RecipientGroup, error)
	GetGroup(id int64) (*model.RecipientGroup, error)
	ListGroups(page, pageSize int) ([]model.RecipientGroup, int64, error)
	UpdateGroup(id int64, name string, description *string, rules []model.RecipientGroupRule) (*model.RecipientGroup, error)
	DeleteGroup(id int64) error

	AddMembersToStaticGroup(groupID int64, recipientIDs []int64) error
	RemoveMembersFromStaticGroup(groupID int64, recipientIDs []int64) error

	CountRecipients(groupID int64, limit *int, offset *int) (int, error)
	ResolveRecipients(groupID int64, searchAfter []interface{}, pageSize int, limit *int, offset *int) ([]*model.Recipient, []interface{}, error)
	PreviewDynamicGroup(rules []model.RecipientGroupRule) (int64, error)
}

type recipientGroupService struct {
	groupRepo     repository.RecipientGroupRepository
	recipientRepo repository.RecipientRepository
}

// NewRecipientGroupService creates a new RecipientGroupService.
func NewRecipientGroupService(groupRepo repository.RecipientGroupRepository, recipientRepo repository.RecipientRepository) RecipientGroupService {
	return &recipientGroupService{
		groupRepo:     groupRepo,
		recipientRepo: recipientRepo,
	}
}

// CreateGroup creates a new recipient group.
func (s *recipientGroupService) CreateGroup(group *model.RecipientGroup) (*model.RecipientGroup, error) {
	// Validate group type
	if group.GroupType != "static" && group.GroupType != "dynamic" {
		return nil, fmt.Errorf("invalid group type: must be 'static' or 'dynamic'")
	}
	// Dynamic groups cannot have members at creation, and vice-versa
	if group.GroupType == "dynamic" && len(group.Members) > 0 {
		return nil, fmt.Errorf("dynamic groups cannot have direct members")
	}
	if group.GroupType == "static" && len(group.Rules) > 0 {
		return nil, fmt.Errorf("static groups cannot have rules")
	}

	if err := s.groupRepo.Create(group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	return group, nil
}

// GetGroup retrieves a single recipient group by its ID.
func (s *recipientGroupService) GetGroup(id int64) (*model.RecipientGroup, error) {
	return s.groupRepo.FindByID(id)
}

// ListGroups retrieves a paginated list of recipient groups.
func (s *recipientGroupService) ListGroups(page, pageSize int) ([]model.RecipientGroup, int64, error) {
	return s.groupRepo.List(page, pageSize)
}

// UpdateGroup updates an existing recipient group.
func (s *recipientGroupService) UpdateGroup(id int64, name string, description *string, rules []model.RecipientGroupRule) (*model.RecipientGroup, error) {
	group, err := s.groupRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("group with id %d not found", id)
	}

	if group.GroupType == "static" && len(rules) > 0 {
		return nil, errors.New("cannot add rules to a static group")
	}

	group.Name = name
	group.Description = description
	group.Rules = rules // Replace existing rules

	if err := s.groupRepo.Update(group); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}
	return group, nil
}

// DeleteGroup deletes a recipient group.
func (s *recipientGroupService) DeleteGroup(id int64) error {
	return s.groupRepo.Delete(id)
}

// AddMembersToStaticGroup adds recipients to a static group.
func (s *recipientGroupService) AddMembersToStaticGroup(groupID int64, recipientIDs []int64) error {
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return fmt.Errorf("group with id %d not found", groupID)
	}
	if group.GroupType != "static" {
		return fmt.Errorf("can only add members to static groups")
	}
	return s.groupRepo.AddMembers(groupID, recipientIDs)
}

// RemoveMembersFromStaticGroup removes recipients from a static group.
func (s *recipientGroupService) RemoveMembersFromStaticGroup(groupID int64, recipientIDs []int64) error {
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		return fmt.Errorf("group with id %d not found", groupID)
	}
	if group.GroupType != "static" {
		return fmt.Errorf("can only remove members from static groups")
	}
	return s.groupRepo.RemoveMembers(groupID, recipientIDs)
}

// ResolveRecipients is the core engine that gets the final recipient list for a task.
// It now supports pagination to handle large recipient groups efficiently.
func (s *recipientGroupService) ResolveRecipients(groupID int64, searchAfter []interface{}, pageSize int, limit *int, offset *int) ([]*model.Recipient, []interface{}, error) {
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("recipient group with id %d not found", groupID)
		}
		return nil, nil, fmt.Errorf("failed to retrieve group %d: %w", groupID, err)
	}

	if group.GroupType == "static" {
		// TODO: Static groups currently use offset-based pagination which has limitations at scale.
		// A more robust solution would involve keyset pagination on the SQL database.
		// For now, we adapt the searchAfter (page number) for compatibility.
		page := 1
		if offset != nil && *offset > 0 {
			page = (*offset / pageSize) + 1
		}

		recipients, err := s.recipientRepo.FindByGroupID(groupID, page, pageSize)
		if err != nil {
			return nil, nil, err
		}
		var nextSearchAfter []interface{}
		if len(recipients) > 0 {
			nextSearchAfter = []interface{}{float64(page + 1)}
		}
		return recipients, nextSearchAfter, nil
	}

	if group.GroupType == "dynamic" {
		if len(group.Rules) == 0 {
			// A dynamic group with no rules should return no one.
			return []*model.Recipient{}, nil, nil
		}
		// The magic happens here: resolving recipients based on stored rules with search_after pagination.
		return s.groupRepo.FindRecipientsByRules(group.Rules, searchAfter, pageSize, limit, offset)
	}

	return nil, nil, fmt.Errorf("unknown group type '%s' for group %d", group.GroupType, groupID)
}

// CountRecipients returns the total count of recipients in a group (static or dynamic).
func (s *recipientGroupService) CountRecipients(groupID int64, limit *int, offset *int) (int, error) {
	group, err := s.groupRepo.FindByID(groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("recipient group with id %d not found", groupID)
		}
		return 0, fmt.Errorf("failed to retrieve group %d: %w", groupID, err)
	}

	if group.GroupType == "static" {
		// For static groups, we need to count the members directly
		// This could be optimized with a dedicated repository method
		count, err := s.recipientRepo.CountByGroupID(groupID)
		if err != nil {
			return 0, fmt.Errorf("failed to count static group members: %w", err)
		}
		return int(count), nil
	}

	if group.GroupType == "dynamic" {
		if len(group.Rules) == 0 {
			return 0, nil
		}
		count, err := s.groupRepo.CountByRules(group.Rules, limit, offset)
		if err != nil {
			return 0, fmt.Errorf("failed to count dynamic group members: %w", err)
		}
		return int(count), nil
	}

	return 0, fmt.Errorf("unknown group type '%s' for group %d", group.GroupType, groupID)
}

// PreviewDynamicGroup returns the count of recipients that match a given set of rules.
// It now directly gets the count from the repository, avoiding fetching all recipients.
func (s *recipientGroupService) PreviewDynamicGroup(rules []model.RecipientGroupRule) (int64, error) {
	return s.groupRepo.CountByRules(rules, nil, nil)
}
