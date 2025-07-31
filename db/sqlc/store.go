package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	UpvoteCommentTx(ctx context.Context, arg UpvoteCommentTxParams) (UpvoteCommentTxResult, error)
	DownvoteCommentTx(ctx context.Context, arg DownvoteCommentTxParams) (DownvoteCommentTxResult, error)
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
