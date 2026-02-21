package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const opSoftDeleteUser = "soft-delete-user"

// SoftDeleteUserTxResult consists of fields only relevant to the delete operation.
type SoftDeleteUserTxResult struct {
	ID             int64       `json:"id"`
	DisplayName    string      `json:"display_name"`
	Username       string      `json:"username"`
	Email          string      `json:"email"`
	ProfileImgURL  pgtype.Text `json:"profile_img_url"`
	IsDeleted      bool        `json:"is_deleted"`
	DeletedAt      time.Time   `json:"deleted_at"`
	LastModifiedAt time.Time   `json:"last_modified_at"`
}

// SoftDeleteUserTx deletes the user's auth sessions and WebAuthn credentials,
// then marks the user as soft-deleted, all within a single transaction.
// Returns KindNotFound if the user does not exist, or KindInternal on database
// errors or unexpected failure.
func (store *SQLStore) SoftDeleteUserTx(ctx context.Context, userID int64) (SoftDeleteUserTxResult, error) {
	var result SoftDeleteUserTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error
		err = q.deleteUserSessions(ctx, userID)
		if err != nil {
			return sqlError(
				opSoftDeleteUser,
				opDetails{userID: userID, entity: entSession},
				err,
			)
		}

		err = q.deleteUserCredentials(ctx, userID)
		if err != nil {
			return sqlError(
				opSoftDeleteUser,
				opDetails{userID: userID, entity: entWauthnCred},
				err,
			)
		}

		row, err := q.softDeleteUser(ctx, userID)

		if err != nil {
			// if 'not found' return error since we cannot return missing user's details
			if errors.Is(err, pgx.ErrNoRows) {
				return notFoundError(opSoftDeleteUser, entUser, userID)
			}

			// else check for sql errors
			return sqlError(
				opSoftDeleteUser,
				opDetails{userID: userID, entity: entUser},
				err,
			)
		}

		// happy case
		if row.Success {
			result = SoftDeleteUserTxResult{
				ID:             row.ID,
				DisplayName:    row.DisplayName,
				Username:       row.Username,
				Email:          row.Email,
				ProfileImgURL:  row.ProfileImgUrl,
				IsDeleted:      row.IsDeleted,
				DeletedAt:      row.DeletedAt,
				LastModifiedAt: row.LastModifiedAt,
			}

			return nil
		}

		return newOpError(
			opSoftDeleteUser,
			KindInternal,
			entUser,
			fmt.Errorf("failed to delete user with id %d", userID),
			withEntityID(userID),
		)
	})

	return result, err
}
