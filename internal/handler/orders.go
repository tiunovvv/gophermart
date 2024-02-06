package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tiunovvv/gophermart/internal/config"
	"github.com/tiunovvv/gophermart/internal/mart"
	"go.uber.org/zap"

	myErrors "github.com/tiunovvv/gophermart/internal/errors"
)

type Handler struct {
	cfg  *config.Config
	mart *mart.Mart
	log  *zap.SugaredLogger
}

func NewHandler(cfg *config.Config, mart *mart.Mart, log *zap.SugaredLogger) *Handler {
	return &Handler{
		cfg:  cfg,
		mart: mart,
		log:  log,
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
	if !h.mart.CheckLunaAlgorithm(number) {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
		return
	}

	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = h.mart.SaveOrder(c, userID, number)

	if errors.Is(err, myErrors.ErrOrderSavedByThisUser) {
		c.Status(http.StatusOK)
		return
	}

	if errors.Is(err, myErrors.ErrOrderSavedByOtherUser) {
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusAccepted)
}

func (h *Handler) GetOrders(c *gin.Context) {
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	orders, err := h.mart.GetOrdersForUser(c, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *Handler) GetBalance(c *gin.Context) {
	userID := h.getUserID(c)
	if len(userID) == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	balance, err := h.mart.GetBalance(c, userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, balance)
}
