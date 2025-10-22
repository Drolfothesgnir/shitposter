package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	Shutdown() // Graceful DB shutdown.
	CreateUserWithCredentialsTx(ctx context.Context, arg CreateUserWithCredentialsTxParams) (CreateUserWithCredentialsTxResult, error)
	SoftDeleteUserTx(ctx context.Context, userID int64) error
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
