package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tiunovvv/gophermart/internal/middleware"
)

func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()

	router.Use(middleware.GinLogger(h.log))
	const seconds = 5 * time.Second
	router.Use(middleware.GinTimeOut(seconds, "timeout error"))

	router.POST("/api/user/register", h.Register)
	router.POST("/api/user/login", h.Login)

	authGroup := router.Group("/api/user").Use(middleware.RequireAuth)

	authGroup.POST("orders", h.SaveOrder)
	authGroup.POST("balance/withdraw", h.SaveWithdraw)

	authGroup.GET("orders", h.GetOrders)
	authGroup.GET("balance", h.GetBalance)
	authGroup.GET("withdrawals", h.GetWithdrawals)

	return router
}
