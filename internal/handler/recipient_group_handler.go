// Package handler contains the HTTP handlers for the application.
//
// import "email-service/internal/handler"
package handler

import (
	"email-service/internal/model"
	"email-service/internal/service"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RecipientGroupHandler handles HTTP requests related to recipient groups.
type RecipientGroupHandler struct {
	service service.RecipientGroupService
}

// NewRecipientGroupHandler creates a new RecipientGroupHandler.
func NewRecipientGroupHandler(s service.RecipientGroupService) *RecipientGroupHandler {
	return &RecipientGroupHandler{service: s}
}

// CreateGroupRequest represents the request body for creating a recipient group.
type CreateGroupRequest struct {
	Name        string                     `json:"name" binding:"required"`
	Description *string                    `json:"description"`
	GroupType   string                     `json:"group_type" binding:"required,oneof=static dynamic"`
	Rules       []model.RecipientGroupRule `json:"rules"`
	MemberIDs   []int64                    `json:"member_ids"`
}

// CreateRecipientGroup handles the creation of a new recipient group.
// @Summary Create a recipient group
// @Description Creates either a static or a dynamic recipient group. For static groups, provide member_ids. For dynamic groups, provide rules.
// @Tags Recipient Groups
// @Accept  json
// @Produce  json
// @Param   group  body      CreateGroupRequest  true  "Group Creation Request"
// @Success 201 {object} Response{data=model.RecipientGroup} "Successfully created group"
// @Failure 400 {object} Response "Invalid request body or parameters"
// @Failure 500 {object} Response "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/recipient-groups [post]
func (h *RecipientGroupHandler) CreateRecipientGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	userID, err := GetUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, Response{Error: err.Error()})
		return
	}

	group := &model.RecipientGroup{
		Name:            req.Name,
		Description:     req.Description,
		GroupType:       req.GroupType,
		CreatedByUserID: userID,
		Rules:           req.Rules,
	}

	createdGroup, err := h.service.CreateGroup(group)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	// If it's a static group and has members, add them.
	if createdGroup.GroupType == "static" && len(req.MemberIDs) > 0 {
		if err := h.service.AddMembersToStaticGroup(createdGroup.ID, req.MemberIDs); err != nil {
			log.Printf("Group %d created, but failed to add members: %v", createdGroup.ID, err)
		}
	}

	c.JSON(http.StatusCreated, Response{Data: createdGroup})
}

// ListRecipientGroups handles listing all recipient groups.
// @Summary List recipient groups
// @Description Retrieves a paginated list of all recipient groups.
// @Tags Recipient Groups
// @Produce  json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(10)
// @Success 200 {object} PaginatedResponse{data=[]model.RecipientGroup} "List of groups"
// @Failure 500 {object} Response "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/recipient-groups [get]
func (h *RecipientGroupHandler) ListRecipientGroups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	groups, total, err := h.service.ListGroups(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Data: groups,
		Pagination: Pagination{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetRecipientGroup handles retrieving a single recipient group.
// @Summary Get a recipient group
// @Description Retrieves details of a specific recipient group by its ID.
// @Tags Recipient Groups
// @Produce  json
// @Param id path int true "Group ID"
// @Success 200 {object} Response{data=model.RecipientGroup} "Group details"
// @Failure 400 {object} Response "Invalid request body or parameters"
// @Failure 404 {object} Response "Group not found"
// @Failure 500 {object} Response "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/recipient-groups/{id} [get]
func (h *RecipientGroupHandler) GetRecipientGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "invalid group ID"})
		return
	}

	group, err := h.service.GetGroup(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{Error: "group not found"})
		} else {
			c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, Response{Data: group})
}

// UpdateGroupRequest represents the request body for updating a recipient group.
type UpdateGroupRequest struct {
	Name        string                     `json:"name" binding:"required"`
	Description *string                    `json:"description"`
	Rules       []model.RecipientGroupRule `json:"rules"`
}

// UpdateRecipientGroup handles the update of a recipient group.
// @Summary Update a recipient group
// @Description Updates a recipient group's details. For dynamic groups, rules can be updated.
// @Tags Recipient Groups
// @Accept  json
// @Produce  json
// @Param id path int true "Group ID"
// @Param   group  body      UpdateGroupRequest  true  "Group Update Request"
// @Success 200 {object} Response{data=model.RecipientGroup} "Successfully updated group"
// @Failure 400 {object} Response "Invalid request body or parameters"
// @Failure 404 {object} Response "Group not found"
// @Failure 500 {object} Response "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/recipient-groups/{id} [put]
func (h *RecipientGroupHandler) UpdateRecipientGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "invalid group ID"})
		return
	}

	var req UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	updatedGroup, err := h.service.UpdateGroup(id, req.Name, req.Description, req.Rules)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{Error: "group not found"})
		} else {
			c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, Response{Data: updatedGroup})
}

// DeleteRecipientGroup handles the deletion of a recipient group.
// @Summary Delete a recipient group
// @Description Deletes a recipient group and its associations.
// @Tags Recipient Groups
// @Produce  json
// @Param id path int true "Group ID"
// @Success 204 "Successfully deleted group"
// @Failure 400 {object} Response "Invalid group ID"
// @Failure 404 {object} Response "Group not found"
// @Failure 500 {object} Response "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/recipient-groups/{id} [delete]
func (h *RecipientGroupHandler) DeleteRecipientGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "invalid group ID"})
		return
	}

	err = h.service.DeleteGroup(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, Response{Error: "group not found"})
		} else {
			c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ModifyMembersRequest represents the request body for adding/removing members from a static group.
type ModifyMembersRequest struct {
	MemberIDs []int64 `json:"member_ids" binding:"required"`
}

// AddRecipientGroupMembers handles adding members to a static recipient group.
// @Summary Add members to a static group
// @Description Adds a list of recipients to a static group.
// @Tags Recipient Groups
// @Accept  json
// @Produce  json
// @Param id path int true "Group ID"
// @Param members body ModifyMembersRequest true "Members to Add"
// @Success 200 {object} Response "Successfully added members"
// @Failure 400 {object} Response "Invalid request body or parameters"
// @Failure 404 {object} Response "Group not found"
// @Failure 500 {object} Response "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/recipient-groups/{id}/members [post]
func (h *RecipientGroupHandler) AddRecipientGroupMembers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "invalid group ID"})
		return
	}

	var req ModifyMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	err = h.service.AddMembersToStaticGroup(id, req.MemberIDs)
	if err != nil {
		// More specific error handling could be added here based on service layer errors
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Data: "members added successfully"})
}

// RemoveRecipientGroupMembers handles removing members from a static recipient group.
// @Summary Remove members from a static group
// @Description Removes a list of recipients from a static group.
// @Tags Recipient Groups
// @Accept  json
// @Produce  json
// @Param id path int true "Group ID"
// @Param members body ModifyMembersRequest true "Members to Remove"
// @Success 200 {object} Response "Successfully removed members"
// @Failure 400 {object} Response "Invalid request body or parameters"
// @Failure 404 {object} Response "Group not found"
// @Failure 500 {object} Response "Internal server error"
// @Security ApiKeyAuth
// @Router /api/v1/recipient-groups/{id}/members [delete]
func (h *RecipientGroupHandler) RemoveRecipientGroupMembers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "invalid group ID"})
		return
	}

	var req ModifyMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	err = h.service.RemoveMembersFromStaticGroup(id, req.MemberIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Data: "members removed successfully"})
}

// GetUserIDFromContext retrieves the user ID from the Gin context.
func GetUserIDFromContext(c *gin.Context) (int64, error) {
	val, exists := c.Get("userID")
	if !exists {
		return 0, errors.New("userID not found in context")
	}

	userID, ok := val.(int64)
	if !ok {
		return 0, errors.New("userID in context is not of type int64")
	}

	return userID, nil
}
