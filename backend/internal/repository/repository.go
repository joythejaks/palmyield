// Package repository wraps sqlc-generated queries with the shared pgx pool.
package repository

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joythejaks/palmyield/backend/internal/repository/sqlcgen"
)

// Repository exposes generated query methods plus the underlying pool for
// callers that need an explicit transaction (e.g. the idempotent harvest
// sync batch insert).
type Repository struct {
	*sqlcgen.Queries
	Pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		Queries: sqlcgen.New(pool),
		Pool:    pool,
	}
}
