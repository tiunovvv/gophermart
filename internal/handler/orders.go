package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tiunovvv/gophermart/internal/config"
	"github.com/tiunovvv/gophermart/internal/mart"
	"github.com/tiunovvv/gophermart/internal/models"
	"go.uber.org/zap"
)

type Handler struct {
	config *config.Config
	mart   *mart.Mart
	logger *zap.Logger
}

func NewHandler(config *config.Config, mart *mart.Mart, logger *zap.Logger) *Handler {
	return &Handler{
		config: config,
		mart:   mart,
		logger: logger,
	}
}

func (h *Handler) SaveOrder(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	number := string(body)
	if !h.mart.CheckLuhnAlgorithm(number) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	userIDDB, err := h.mart.GetUserIDForOrder(c, number)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if userIDDB == userID {
		c.AbortWithStatus(http.StatusOK)
		return
	}
	if len(userIDDB) != 0 {
		c.AbortWithStatus(http.StatusConflict)
		return
	}
	h.mart.SaveOrder(c, userID, number)
	c.AbortWithStatus(http.StatusAccepted)
}

func (h *Handler) GetOrders(c *gin.Context) {
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	numbers, err := h.mart.GetNumbersForUser(c, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(numbers) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}

	orders := make([]models.OrderWithTime, 0)
	for number, time := range numbers {
		order, err := h.mart.GetOrderInfo(h.config.AccrualSystemAddress, number)
		if err != nil {
			h.logger.Sugar().Errorf("failed to get info about order: %s, %w", number, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if len(order.Number) == 0 {
			order.Number = number
		}
		order.UploadedAt = time
		orders = append(orders, order)
	}
	c.AbortWithStatusJSON(http.StatusOK, orders)
}
