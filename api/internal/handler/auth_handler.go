package handler

import (
	"net/http"

	domainuser "contract-scanner/internal/domain/user"
	"contract-scanner/internal/middleware"
	"contract-scanner/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	registerUser usecase.IRegisterUser
	loginUser    usecase.ILoginUser
}

func NewAuthHandler(registerUser usecase.IRegisterUser, loginUser usecase.ILoginUser) *AuthHandler {
	return &AuthHandler{
		registerUser: registerUser,
		loginUser:    loginUser,
	}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type authResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// Register handles POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.registerUser.Execute(c.Request.Context(), usecase.RegisterUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch err {
		case domainuser.ErrEmailAlreadyExists, domainuser.ErrUsernameAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, authResponse{
		Token:    output.Token,
		UserID:   output.UserID.String(),
		Username: output.Username,
	})
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.loginUser.Execute(c.Request.Context(), usecase.LoginUserInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if err == domainuser.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResponse{
		Token:    output.Token,
		UserID:   output.UserID.String(),
		Username: output.Username,
	})
}

// Me handles GET /api/auth/me — returns the authenticated user info from the JWT claims
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	username, _ := c.Get(middleware.UsernameKey)

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
	})
}
