package handler

import (
	_ "email-service/internal/model" // Blank import for swag to find model types
	"email-service/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related HTTP requests.
type UserHandler struct {
	userService service.UserService
}

// NewUserHandler creates a new instance of UserHandler.
func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type createUserRequest struct {
	Username string `json:"username" binding:"required" example:"newuser"`
	Email    string `json:"email" binding:"required,email" example:"new.user@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"strongpassword"`
	IsAdmin  bool   `json:"is_admin" example:"false"`
}

// CreateUser handles the creation of a new user.
// @Summary      Create a new user
// @Description  Creates a new user account. This is a protected endpoint.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      createUserRequest  true  "User details"
// @Success      201   {object}  Response{data=model.User}  "Successfully created user"
// @Failure      400   {object}  Response      "Invalid request body"
// @Failure      409   {object}  Response      "User with this username or email already exists"
// @Failure      500   {object}  Response      "Internal server error"
// @Security     ApiKeyAuth
// @Router       /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: err.Error()})
		return
	}

	// In a real app, you would check if the current user (from context) is an admin
	// before allowing setting IsAdmin=true.
	// For now, we'll allow it.

	user, err := h.userService.CreateUser(req.Username, req.Email, req.Password, req.IsAdmin)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			c.JSON(http.StatusConflict, Response{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to create user"})
		return
	}

	// The user model's JSON tags should prevent the password hash from being marshaled.
	// We return the user object wrapped in our standard response.
	c.JSON(http.StatusCreated, Response{Data: user})
}
