package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	// Shutdown shuts down the db connection pool gracefully.
	Shutdown()

	// CreateUserWithCredentialsTx creates both user and their credentials in one transaction.
	CreateUserWithCredentialsTx(ctx context.Context, arg CreateUserWithCredentialsTxParams) (CreateUserWithCredentialsTxResult, error)

	// GetUser retrieves data of user with provided ID.
	GetUser(ctx context.Context, userID int64) (User, error)

	// SoftDeleteUserTx deletes user's auth sessions, webauthn credentials and soft deletes the user.
	SoftDeleteUserTx(ctx context.Context, userID int64) error

	// InsertCommentTx creates a new comment, either root or reply.
	//
	// Errors returned:
	// 	 - ErrParentCommentNotFound       - when trying to reply to a not existing comment
	//   - ErrParentCommentPostIDMismatch - if the reply's parent post ID and provided post ID mismatch
	//   - ErrParentCommentDeleted        - when trying to reply to a deleted comment
	//   - ErrInvalidPostID               - when trying to comment a missing post
	//
	// May also return database or transaction errors.
	InsertCommentTx(ctx context.Context, arg InsertCommentTxParams) (Comment, error)

	// QueryComments returns paginated set of comments ordered by popularity or date (old/new).
	QueryComments(ctx context.Context, query CommentQuery) ([]CommentsWithAuthor, error)

	UpdateComment(ctx context.Context, arg UpdateCommentParams) (UpdateCommentResult, error)

	// DeleteCommentTx deletes a comment or soft-deletes it if it has children.
	//
	// Errors returned:
	//   - ErrEntityNotFound            – if the target comment does not exist
	//   - ErrEntityDoesNotBelongToUser – if the comment belongs to another user
	//   - ErrInvalidPostID             – if post_id mismatch happens
	//   - ErrDataCorrupted             – unexpected inconsistent DB state
	//
	// May also return database or transaction errors.
	DeleteCommentTx(ctx context.Context, arg DeleteCommentTxParams) (DeleteCommentTxResult, error)

	// VoteCommentTx performs voting for comment.
	//
	// Errors returned:
	//   - ErrInvalidVoteValue - if provided vote is not 1 or -1
	//   - ErrInvalidCommentID - if the user tries to vote for a non existent comment
	//   - ErrInvalidUserID    - if the voting user does not exist
	//   - ErrDuplicateVote    - when voting is repeated
	//   - ErrEntityDeleted    - when trying to vote for deleted comment
	//
	// May also return database or transaction errors.
	VoteCommentTx(ctx context.Context, arg VoteCommentTxParams) (Comment, error)
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
