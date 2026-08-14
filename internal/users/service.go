package users

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service contains the business logic for Norest user operations.
type Service struct {
	repo *Repository
}

// NewService creates a new user service.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo: NewRepository(pool),
	}
}
