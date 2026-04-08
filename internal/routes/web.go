package routes

import (
	"FoPQer/go-fermart/internal/handlers"

	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

type Routes struct {
	OrderHandler   *handlers.OrderHandler
	BalanceHandler *handlers.BalanceHandler
	AuthHandler    *handlers.AuthHandler
}

var secret_key = []byte("your-secret-key")

func (r *Routes) SetupRoutes() *echo.Echo {
	e := echo.New()
	
	e.Use(middleware.RequestLogger()) 
  	e.Use(middleware.Recover())

	user := e.Group("/api/user")
	user.POST("/register", r.AuthHandler.Register)
	user.POST("/login", r.AuthHandler.Login)

	auth := user.Group("", echojwt.JWT(secret_key))
	auth.POST("/orders", r.OrderHandler.LoadOrder)
	auth.GET("/orders", r.OrderHandler.GetOrders)
	auth.GET("/balance", r.BalanceHandler.GetBalance)
	auth.POST("/balance/withdraw", r.BalanceHandler.Withdraw)
	auth.GET("/withdrawals", r.BalanceHandler.GetWithdrawals)

	order := e.Group("/api/orders")
	order.GET("/:id", r.OrderHandler.GetOrders)

	return e
}