package repository

import (
	"FoPQer/go-fermart/internal/repository/order"
	"FoPQer/go-fermart/internal/repository/user"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Factory struct {
	conn *pgxpool.Pool
}

func NewFactory(conn *pgxpool.Pool) *Factory {
	return &Factory{conn: conn}
}

func (f *Factory) GetUserRepository() user.Repository {
	var repo user.Repository
	if f.conn != nil {
		log.Println("Using PostgreSQL user repository")
		repo = user.NewPgsqlRepository(f.conn)
	} else {
		log.Println("Using in-memory user repository")
		repo = user.NewMemoryRepository()
	}
	return repo
}

func (f *Factory) GetOrderRepository() order.Repository {
	var repo order.Repository
	if f.conn != nil {
		log.Println("Using PostgreSQL order repository")
		repo = order.NewPgsqlRepository(f.conn)
	} else {
		log.Println("Using in-memory order repository")
		repo = order.NewMemoryRepository()
	}
	return repo
}