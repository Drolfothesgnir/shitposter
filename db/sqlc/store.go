package db

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	Shutdown() // Graceful DB shutdown.
}

type SQLStore struct {
	*Queries
	connPool *pgxpool.Pool
}

func NewStore(connPool *pgxpool.Pool) Store {
	return &SQLStore{
		connPool: connPool,
		Queries:  New(connPool),
	}
}

func (store *SQLStore) Shutdown() {
	store.connPool.Close()
}
