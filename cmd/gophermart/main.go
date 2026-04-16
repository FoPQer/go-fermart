package main

import (
	"FoPQer/go-fermart/internal/auth"
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/handlers"
	"FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/repository/user"
	"FoPQer/go-fermart/internal/routes"
	"FoPQer/go-fermart/internal/services"
	"log/slog"
)

func main() {
	config := config.NewConfig()
	config.Load()
	userRepository := user.NewMemoryRepository()
	orderRepository := order.NewMemoryRepository()

	orderService := services.NewOrderService(orderRepository)
	userService := services.NewUserService(userRepository)
	claimsService := auth.NewClaimsService()

	orderHandler := handlers.NewOrderHandler(orderService)
	balanceHandler := handlers.NewBalanceHandler(userService)
	authHandler := handlers.NewAuthHandler(userService, claimsService, config)

	r := routes.NewRoutes(orderHandler, balanceHandler, authHandler, config)
	e := r.SetupRoutes()

	// Start server
	if err := e.Start(":8080"); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
