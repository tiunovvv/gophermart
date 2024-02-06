package handler

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	myErrors "github.com/tiunovvv/gophermart/internal/errors"
	"github.com/tiunovvv/gophermart/internal/models"
)

const maxAge = 3600 * 24 * 30

func (h *Handler) Register(c *gin.Context) {
	h.authenticateUser(c, "register")
}

func (h *Handler) Login(c *gin.Context) {
	h.authenticateUser(c, "login")
}

func (h *Handler) authenticateUser(c *gin.Context, action string) {
	var user models.User

	if err := c.ShouldBindJSON(&user); err != nil {
		h.log.Error("failed to decode request JSON body: %w", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var userID string
	var err error
	switch action {
	case "register":
		userID, err = h.mart.NewUser(c, user)
		if errors.Is(err, myErrors.ErrLoginAlreadySaved) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}
	case "login":
		userID, err = h.mart.GetUserID(c, user)
		if err != nil {
			h.log.Error("failed to login: %w", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}

	token, err := getToken(userID)
	if err != nil {
		h.log.Error("failed to create token: %w", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.SetCookie("Authorization", token, maxAge, "", "", false, true)
	c.Status(http.StatusOK)
}

func getToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
	})

	tkn, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		return "", fmt.Errorf("failed get sign: %w", err)
	}
	return tkn, nil
}

func (h *Handler) getUserID(c *gin.Context) string {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	userID, ok := userIDInterface.(string)
	if !ok {
		h.log.Errorf("failed to get userID from %v", userIDInterface)
		return ""
	}
	return userID
}
