package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	myErrors "github.com/tiunovvv/gophermart/internal/errors"
	"github.com/tiunovvv/gophermart/internal/models"
)

func (h *Handler) Balance(c *gin.Context) {
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	sumOrders, err := h.mart.GetSumOrders(c, h.config.AccrualSystemAddress, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	var balance models.Balance
	withdrawn, err := h.mart.GetSumWithdrawn(c, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	balance.Current = sumOrders - withdrawn
	balance.Withdrawn = withdrawn

	c.AbortWithStatusJSON(http.StatusOK, balance)
}

func (h *Handler) Withdraw(c *gin.Context) {
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var withdrawNew models.Withdraw
	if err := c.ShouldBindJSON(&withdrawNew); err != nil {
		h.logger.Sugar().Error("failed to decode request JSON body: %w", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !h.mart.CheckLuhnAlgorithm(withdrawNew.Order) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	sumOrders, err := h.mart.GetSumOrders(c, h.config.AccrualSystemAddress, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	sumWithdraw, err := h.mart.GetSumWithdrawn(c, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if sumOrders-sumWithdraw-withdrawNew.Sum < 0 {
		c.AbortWithStatus(http.StatusPaymentRequired)
		return
	}

	if err := h.mart.SaveWithdraw(c, userID, withdrawNew.Order, withdrawNew.Sum); err != nil {
		if errors.Is(err, myErrors.ErrWithdrawAlreadySaved) {
			c.AbortWithStatus(http.StatusUnprocessableEntity)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

func (h *Handler) Withdrawals(c *gin.Context) {
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

	c.AbortWithStatusJSON(http.StatusOK, windrawals)
}
