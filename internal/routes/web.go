package routes

import (
	"FoPQer/go-fermart/internal/handlers"
	"FoPQer/go-fermart/internal/middlewares"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type Routes struct {
	OrderHandler   *handlers.OrderHandler
	BalanceHandler *handlers.BalanceHandler
	AuthHandler    *handlers.AuthHandler
}

func (r *Routes) SetupRoutes() *echo.Echo {
	e := echo.New()
	
	e.Use(middleware.RequestLogger()) 
  	e.Use(middleware.Recover())

	user := e.Group("/api/user")

	user.POST("/register", r.AuthHandler.Register)
	user.POST("/login", r.AuthHandler.Login)
	user.POST("/orders", r.OrderHandler.LoadOrder, middlewares.WithAuth())
	user.GET("/orders", r.OrderHandler.GetOrders, middlewares.WithAuth())
	user.GET("/balance", r.BalanceHandler.GetBalance, middlewares.WithAuth())
	user.POST("/balance/withdraw", r.BalanceHandler.Withdraw, middlewares.WithAuth())
	user.GET("/withdrawals", r.BalanceHandler.GetWithdrawals, middlewares.WithAuth())

	order := e.Group("/api/orders")
	order.GET("/:id", r.OrderHandler.GetOrders)

	return e
}