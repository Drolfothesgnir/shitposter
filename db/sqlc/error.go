package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type Kind int

const (
	// KindInternal represents an unexpected internal failure such as
	// driver errors, transaction aborts, or unknown DB failures.
	KindInternal Kind = iota

	// KindNotFound indicates that a referenced entity does not exist.
	KindNotFound

	// KindInvalid indicates invalid input or parameters.
	KindInvalid

	// KindPermission indicates that the caller lacks permission
	// to perform the operation.
	KindPermission

	// KindRelation indicates that the referenced entities do not relate
	// to each other logically (e.g., parent comment belongs to a different post).
	KindRelation

	// KindConflict indicates a uniqueness or duplication conflict.
	KindConflict

	// KindDeleted indicates that the entity exists but is soft-deleted.
	KindDeleted

	// KindCorrupted indicates an inconsistent or impossible DB state.
	KindCorrupted
)

var kindNames = map[Kind]string{
	KindInternal:   "internal",
	KindNotFound:   "not_found",
	KindInvalid:    "invalid",
	KindPermission: "permission",
	KindRelation:   "relation",
	KindConflict:   "conflict",
	KindDeleted:    "deleted",
	KindCorrupted:  "corrupted",
}

func (k Kind) String() string {
	if s, ok := kindNames[k]; ok {
		return s
	}
	return "unknown"
}

const (
	entPost        = "post"
	entPostVote    = "post-vote"
	entComment     = "comment"
	entCommentVote = "comment-vote"
	entUser        = "user"
	entWauthnCred  = "webauthn-credential"
	entSession     = "session"
)

// OpError describes a failure of a database operation.
type OpError struct {
	Op              string // logical operation: "insert-comment", "delete-comment", etc.
	Kind            Kind   // classification of the failure
	Entity          string // entity involved: "comment", "post", "user", etc.
	EntityID        int64  // optional ID of the entity involved
	RelatedEntity   string // optional other entity involved, for example parent comment or post
	RelatedEntityID int64  // optional other entity ID
	FailingField    string // optional name of the failing field: "email", "username", etc.
	UserID          int64  // optional acting user
	Err             error  // underlying error
}

func (e *OpError) Error() string {
	parts := []string{
		e.Op + ":",
		"kind=" + e.Kind.String(),
		"entity=" + e.Entity,
	}

	if e.EntityID != 0 {
		parts = append(parts, fmt.Sprintf("entity_id=%d", e.EntityID))
	}
	if e.RelatedEntity != "" {
		parts = append(parts, "related_entity="+e.RelatedEntity)
	}
	if e.RelatedEntityID != 0 {
		parts = append(parts, fmt.Sprintf("related_entity_id=%d", e.RelatedEntityID))
	}
	if e.FailingField != "" {
		parts = append(parts, "field="+e.FailingField)
	}
	if e.UserID != 0 {
		parts = append(parts, fmt.Sprintf("user_id=%d", e.UserID))
	}
	if e.Err != nil {
		parts = append(parts, "err="+e.Err.Error())
	}

	return strings.Join(parts, " ")
}

func (e *OpError) Unwrap() error {
	return e.Err
}

type errDecorator func(*OpError)

// newOpError builds *OpError from basic and optional parts.
func newOpError(op string, kind Kind, entity string, err error, opts ...errDecorator) *OpError {
	be := &OpError{Op: op, Kind: kind, Entity: entity, Err: err}

	for _, dec := range opts {
		dec(be)
	}

	return be
}

// withEntityID augments base OpError with provided entity ID.
func withEntityID(entityID int64) errDecorator {
	return func(be *OpError) { be.EntityID = entityID }
}

// withUser augments base OpError with provided user ID.
func withUser(userID int64) errDecorator {
	return func(be *OpError) { be.UserID = userID }
}

// withRelated augments base OpError with provided related entity info.
func withRelated(entity string, entityID int64) errDecorator {
	return func(be *OpError) {
		be.RelatedEntity = entity
		be.RelatedEntityID = entityID
	}
}

// withField augments base OpError with provided failing field.
func withField(failingField string) errDecorator {
	return func(be *OpError) { be.FailingField = failingField }
}

// notFoundError builds *OpError for the common "entity not found" case.
func notFoundError(op string, entity string, entityID int64) *OpError {
	return newOpError(
		op,
		KindNotFound,
		entity,
		fmt.Errorf("%s with id %d not found", entity, entityID),
		withEntityID(entityID),
	)
}

type opDetails struct {
	userID    int64
	postID    int64
	commentID int64
	input     string
	entity    string
}

// sqlError builds *OpError for wrapping postgres errors.
func sqlError(op string, det opDetails, err error) *OpError {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {

		// check for the foreign key violation
		// usually means that the caller trying to perform
		// operations with entity related to other non-existent entity
		// or operates on behalf of a non-existent user
		if pgError.Code == "23503" {
			switch pgError.ConstraintName {
			// when attempting to create comment for non-existent post
			case "comments_post_id_fkey":
				return newOpError(
					op,
					KindRelation,
					entComment,
					fmt.Errorf("attempt to create comment for a non-existent post with id [%d]: %w", det.postID, pgError),
					withRelated(entPost, det.postID),
				)

			// when attempting to vote for non-existent comment
			case "comment_votes_comment_id_fkey":
				return newOpError(
					op,
					KindRelation,
					entCommentVote,
					fmt.Errorf("attempt to vote for a non-existent comment with id [%d]: %w", det.commentID, pgError),
					withRelated(entComment, det.commentID),
				)

				// attempting to vote as non-existent user
			case "comment_votes_user_id_fkey":
				return newOpError(
					op,
					KindRelation,
					entCommentVote,
					fmt.Errorf("attempt to vote as a non-existent user with id [%d]: %w", det.userID, pgError),
					withRelated(entUser, det.userID),
				)

			default:
				return newOpError(op, KindRelation, det.entity, pgError)
			}
		}

		// check for unique violations
		if pgError.Code == "23505" {
			switch pgError.ConstraintName {
			// when active user with this username exists
			case "uniq_users_username_active":
				return newOpError(
					op,
					KindConflict,
					entUser,
					fmt.Errorf("user with username '%s' exists: %w", det.input, pgError),
					withEntityID(det.userID),
					withField("username"),
				)

			// when active user with this email exists
			case "uniq_users_email_active":
				return newOpError(
					op,
					KindConflict,
					entUser,
					fmt.Errorf("user with email '%s' exists: %w", det.input, pgError),
					withEntityID(det.userID),
					withField("email"),
				)

			default:
				return newOpError(op, KindConflict, det.entity, pgError)
			}
		}
	}

	return newOpError(op, KindInternal, det.entity, err)
}
