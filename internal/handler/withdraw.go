package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tiunovvv/gophermart/internal/models"

	myErrors "github.com/tiunovvv/gophermart/internal/errors"
)

func (h *Handler) SaveWithdraw(c *gin.Context) {
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var withdraw models.Withdraw
	if err := c.ShouldBindJSON(&withdraw); err != nil {
		h.log.Error("failed to decode request JSON body: %w", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !h.mart.CheckLunaAlgorithm(withdraw.Order) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	err := h.mart.SaveWithdraw(c, userID, withdraw)

	if errors.Is(err, myErrors.ErrWithdrawAlreadySaved) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	if errors.Is(err, myErrors.ErrNoMoney) {
		c.AbortWithStatus(http.StatusPaymentRequired)
		return
	}

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) GetWithdrawals(c *gin.Context) {
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	windrawals, err := h.mart.GetWindrawalsForUser(c, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(windrawals) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, windrawals)
}
