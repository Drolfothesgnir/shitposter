package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store defines the interface for all database operations.
// All methods return *[OpError] on failure with a Kind field that classifies the error.
type Store interface {
	// Shutdown closes the underlying connection pool gracefully.
	// It does not return an error; the pool is simply drained.
	Shutdown()

	// CreateUserWithCredentialsTx creates a user and a WebAuthn credential
	// in a single transaction. On error neither the user nor the credential
	// is persisted.
	//
	// Errors returned (*OpError):
	//   - KindConflict – username or email already taken (unique constraint violation)
	//   - KindInternal – database or transaction error
	CreateUserWithCredentialsTx(ctx context.Context, arg CreateUserWithCredentialsTxParams) (User, error)

	// GetUser retrieves the user with the provided ID.
	//
	// Errors returned (*OpError):
	//   - KindNotFound – no user with the given ID exists
	//   - KindDeleted  – user exists but has been soft-deleted
	//   - KindInternal – database error
	GetUser(ctx context.Context, userID int64) (User, error)

	// UpdateUser applies the non-nil fields in arg to the user record.
	// At least one optional field (Username, Email, ProfileImgURL) must be set.
	//
	// Errors returned (*OpError):
	//   - KindInvalid  – all optional fields are nil (nothing to update)
	//   - KindNotFound – no user with the given ID exists
	//   - KindDeleted  – user exists but has been soft-deleted
	//   - KindConflict – new username or email conflicts with an existing active user
	//   - KindInternal – database error or unexpected failure
	UpdateUser(ctx context.Context, arg UpdateUserParams) (UpdateUserResult, error)

	// SoftDeleteUserTx deletes the user's auth sessions and WebAuthn credentials,
	// then marks the user as soft-deleted, all within a single transaction.
	//
	// Errors returned (*OpError):
	//   - KindNotFound – no user with the given ID exists
	//   - KindInternal – database error or unexpected failure
	SoftDeleteUserTx(ctx context.Context, userID int64) (SoftDeleteUserTxResult, error)

	// UsernameExists reports whether an active user with the given username exists.
	//
	// Errors returned (*OpError):
	//   - KindInternal – database error
	UsernameExists(ctx context.Context, username string) (bool, error)

	// EmailExists reports whether an active user with the given email exists.
	//
	// Errors returned (*OpError):
	//   - KindInternal – database error
	EmailExists(ctx context.Context, email string) (bool, error)

	// InsertCommentTx creates a new comment, either a root comment or a reply
	// to an existing comment, within a transaction.
	//
	// Errors returned (*OpError):
	//   - KindNotFound – parent comment does not exist (reply only)
	//   - KindRelation – parent comment belongs to a different post, or the post does not exist
	//   - KindDeleted  – parent comment has been soft-deleted (reply only)
	//   - KindInternal – database or transaction error
	InsertCommentTx(ctx context.Context, arg InsertCommentTxParams) (Comment, error)

	// QueryComments returns a paginated set of comments for a post, ordered by
	// popularity ("pop"), newest first ("new"), or oldest first ("old").
	// Returns an empty slice when the post has no comments or does not exist.
	//
	// Errors returned (*OpError):
	//   - KindInvalid  – the provided order value is not one of "pop", "new", "old"
	//   - KindInternal – database error
	QueryComments(ctx context.Context, query CommentQuery) ([]CommentsWithAuthor, error)

	// UpdateComment updates the body of a comment identified by CommentID.
	// The caller must own the comment and the comment must belong to the given post.
	//
	// Errors returned (*OpError):
	//   - KindNotFound   – comment does not exist
	//   - KindPermission – comment belongs to another user
	//   - KindDeleted    – comment has been soft-deleted
	//   - KindRelation   – comment belongs to a different post
	//   - KindInternal   – database error or unexpected failure
	UpdateComment(ctx context.Context, arg UpdateCommentParams) (UpdateCommentResult, error)

	// DeleteCommentTx deletes a comment. Leaf comments are hard-deleted;
	// comments with children are soft-deleted (body cleared, is_deleted flag set).
	// Already-deleted comments are treated as a successful no-op.
	//
	// Errors returned (*OpError):
	//   - KindNotFound   – comment does not exist
	//   - KindPermission – comment belongs to another user
	//   - KindRelation   – comment belongs to a different post
	//   - KindCorrupted  – unexpected inconsistent database state
	//   - KindInternal   – database or transaction error
	DeleteCommentTx(ctx context.Context, arg DeleteCommentTxParams) (DeleteCommentTxResult, error)

	// VoteCommentTx records an upvote (+1) or downvote (-1) on a comment
	// and updates the comment's popularity counters within a transaction.
	//
	// Errors returned (*OpError):
	//   - KindInvalid  – vote value is not 1 or -1
	//   - KindRelation – comment or user does not exist (foreign key violation)
	//   - KindConflict – the same vote value has already been cast by this user
	//   - KindDeleted  – comment has been soft-deleted
	//   - KindInternal – database or transaction error
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
