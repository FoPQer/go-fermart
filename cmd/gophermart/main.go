package main

import (
	"FoPQer/go-fermart/internal/auth"
	"FoPQer/go-fermart/internal/config"
	"FoPQer/go-fermart/internal/config/db"
	"FoPQer/go-fermart/internal/handlers"
	"FoPQer/go-fermart/internal/repository"
	"FoPQer/go-fermart/internal/routes"
	"FoPQer/go-fermart/internal/services"
	"errors"
	"log/slog"
)

func main() {
	config := config.NewConfig()
	config.Load()
	pgxConf, err := db.InitPgsql(config)
	if errors.Is(err, db.ErrConnNotFound) {
		slog.Info("Database connection string not found, using file or memory repository")
	} else if err != nil {
		slog.Error("Error initializing database: ", "error", err)
		panic(err)
	}
	if pgxConf.GetDBConn() != nil {
		defer pgxConf.GetDBConn().Close()
	} 

	repoFactory := repository.NewFactory(pgxConf.GetDBConn())

	userRepository := repoFactory.GetUserRepository()
	orderRepository := repoFactory.GetOrderRepository()

	userService := services.NewUserService(userRepository)
	orderService := services.NewOrderService(orderRepository, userService, config)
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
