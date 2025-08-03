package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	Querier
	VoteCommentTx(ctx context.Context, arg VoteCommentTxParams) (VoteCommentTxResult, error)
	DeleteCommentVoteTx(ctx context.Context, arg DeleteCommentVoteTxParams) (DeleteCommentVoteTxResult, error)
	VotePostTx(ctx context.Context, arg VotePostTxParams) (VotePostTxResult, error)
	DeletePostVoteTx(ctx context.Context, arg DeletePostVoteTxParams) (DeletePostVoteTxResult, error)
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
