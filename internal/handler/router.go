package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tiunovvv/gophermart/internal/middleware"
)

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()
	router.Use(middleware.GinLogger(h.logger))
	const seconds = 5 * time.Second
	router.Use(middleware.GinTimeOut(seconds, "timeout error"))

	router.POST("/api/user/register", h.Register)
	router.POST("/api/user/login", h.Login)
	router.POST("/api/user/orders", middleware.RequireAuth, h.SaveOrder)
	router.GET("/api/user/orders", middleware.RequireAuth, h.GetOrders)
	router.GET("/api/user/balance", middleware.RequireAuth, h.Balance)
	router.POST("/api/user/balance/withdraw", middleware.RequireAuth, h.Withdraw)
	router.GET("/api/user/withdrawals", middleware.RequireAuth, h.Withdrawals)

	return router
}
