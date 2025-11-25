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
	InsertCommentTx(ctx context.Context, arg InsertCommentTxParams) (Comment, error)
	DeleteCommentTx(ctx context.Context, arg DeleteCommentTxParams) (DeleteCommentTxResult, error)
	QueryComments(ctx context.Context, query CommentQuery) ([]CommentsWithAuthor, error)
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
