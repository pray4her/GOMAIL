package handler

import (
	"email-service/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new instance of AuthHandler.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// loginRequest represents the request body for user login.
type loginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// Login handles user login requests.
// @Summary      User login
// @Description  Authenticates a user and returns a JWT token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      loginRequest     true  "User credentials"
// @Success      200          {object}  Response{data=map[string]string} "{"token": "..."}"
// @Failure      400          {object}  Response    "Invalid request body"
// @Failure      401          {object}  Response    "Invalid credentials"
// @Failure      500          {object}  Response    "Internal server error"
// @Router       /api/v1/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Error: "Invalid request body: " + err.Error()})
		return
	}

	token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, Response{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Error: "Failed to login"})
		return
	}

	c.JSON(http.StatusOK, Response{Data: gin.H{"token": token}})
}
