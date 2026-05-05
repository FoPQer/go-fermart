package main

import (
	"FoPQer/go-fermart/internal/auth"
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/config/db"
	"FoPQer/go-fermart/internal/handlers"
	"FoPQer/go-fermart/internal/repository"
	"FoPQer/go-fermart/internal/routes"
	"FoPQer/go-fermart/internal/services"
	"context"
	"errors"
	"log"
	"log/slog"
)

func main() {
	config := config.Load()
	pgxConn, err := db.InitPgsql(context.Background(), config)
	if errors.Is(err, db.ErrConnNotFound) {
		log.Fatalf("Database connection string not found, using file or memory repository")
	} else if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	if pgxConn != nil {
		defer pgxConn.Close()
	}

	repoFactory := repository.NewFactory(pgxConn)

	userRepository := repoFactory.GetUserRepository()
	orderRepository := repoFactory.GetOrderRepository()

	userService := services.NewUserService(userRepository)
	orderService := services.NewOrderService(orderRepository, userService, config, repoFactory.GetTransactor())
	defer orderService.Close()
	claimsService := auth.NewClaimsService()

	orderHandler := handlers.NewOrderHandler(orderService)
	balanceHandler := handlers.NewBalanceHandler(userService, orderService)
	authHandler := handlers.NewAuthHandler(userService, claimsService, config)

	r := routes.NewRoutes(orderHandler, balanceHandler, authHandler, config)
	e := r.SetupRoutes()
	slog.Info("Starting server on ", "address", config.GetRunAddr())
	slog.Info("Starting accrual on ", "address", config.GetAccrualAddress())

	// Start server
	if err := e.Start(config.GetRunAddr()); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
