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

// baseError creates OpError without optional fields (no entity or user IDs).
func baseError(op, entity string, kind Kind, err error) *OpError {
	return &OpError{
		Op:     op,
		Kind:   kind,
		Entity: entity,
		Err:    err,
	}
}

// withEntityID augments base OpError with provided entity ID.
func withEntityID(be *OpError, entityID int64) *OpError {
	be.EntityID = entityID
	return be
}

// withUserID augments base OpError with provided user ID.
func withUserID(be *OpError, userID int64) *OpError {
	be.UserID = userID
	return be
}

type opDetails struct {
	userID    int64
	postID    int64
	commentID int64
	input     string
	entity    string
}

func sqlError(op string, det opDetails, err error) *OpError {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {

		// check for the foreign key violation
		// usually means that the caller trying to perform
		// operations with entity related to other non-existent entity
		// or operates on behalf of non-existent user
		if pgError.Code == "23503" {
			switch pgError.ConstraintName {
			// when attempting to create comment for non-existent post
			case "comments_post_id_fkey":
				return &OpError{
					Op:       op,
					Kind:     KindRelation,
					Entity:   "post",
					EntityID: det.postID,
					Err:      fmt.Errorf("attempt to create comment for a non-existent post with id [%d]: %w", det.postID, pgError),
				}

			// when attempting to vote for non-existent comment
			case "comment_votes_comment_id_fkey":
				return &OpError{
					Op:       op,
					Kind:     KindRelation,
					Entity:   "comment",
					EntityID: det.commentID,
					Err:      fmt.Errorf("attempt to vote for a non-existent comment with id [%d]: %w", det.commentID, pgError),
				}

				// attempting to vote as non-existent user
			case "comment_votes_user_id_fkey":
				return &OpError{
					Op:       op,
					Kind:     KindRelation,
					Entity:   "user",
					EntityID: det.userID,
					UserID:   det.userID,
					Err:      fmt.Errorf("attempt to vote as a non-existent user with id [%d]: %w", det.userID, pgError),
				}

			default:
				return baseError(op, det.entity, KindRelation, pgError)
			}
		}

		// check for unique violations
		if pgError.Code == "23505" {
			switch pgError.ConstraintName {
			// when active user with this username exists
			case "uniq_users_username_active":
				return baseError(op, "user", KindConflict, fmt.Errorf("user with username '%s' exists: %w", det.input, pgError))
				// when active user with this email exists
			case "uniq_users_email_active":
				return baseError(op, "user", KindConflict, fmt.Errorf("user with email '%s' exists: %w", det.input, pgError))
			default:
				return baseError(op, det.entity, KindConflict, pgError)
			}
		}
	}

	return baseError(op, det.entity, KindInternal, err)
}
