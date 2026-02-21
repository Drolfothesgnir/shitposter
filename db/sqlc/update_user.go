package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Drolfothesgnir/shitposter/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const opUpdateUser = "update-user"

type UpdateUserParams struct {
	ID            int64
	Username      *string
	Email         *string
	ProfileImgURL *string
}

// empty will return true if all optional field are nil.
func (p *UpdateUserParams) empty() bool {
	return p.Username == nil &&
		p.Email == nil &&
		p.ProfileImgURL == nil
}

// UpdateUserResult consists of fields only relevant for the update operation.
type UpdateUserResult struct {
	ID             int64       `json:"id"`
	Username       string      `json:"username"`
	Email          string      `json:"email"`
	ProfileImgURL  pgtype.Text `json:"profile_img_url"`
	LastModifiedAt time.Time   `json:"last_modified_at"`
}

// UpdateUser applies the non-nil fields in arg to the user record.
// Returns KindInvalid if all fields are nil, KindNotFound if the user does not exist,
// KindDeleted if the user is soft-deleted, KindConflict on username/email uniqueness
// violations, or KindInternal on database errors.
func (s *SQLStore) UpdateUser(ctx context.Context, arg UpdateUserParams) (UpdateUserResult, error) {
	if arg.empty() {
		opErr := newOpError(
			opUpdateUser,
			KindInvalid,
			entUser,
			errors.New("all provided fields are empty"),
		)

		return UpdateUserResult{}, opErr
	}

	row, err := s.updateUser(ctx, updateUserParams{
		PUserID:       arg.ID,
		Username:      util.StringToPgxText(arg.Username),
		Email:         util.StringToPgxText(arg.Email),
		ProfileImgUrl: util.StringToPgxText(arg.ProfileImgURL),
	})

	if err != nil {
		// if 'not found' is returned it means the target user doesn't exist
		if errors.Is(err, pgx.ErrNoRows) {
			return UpdateUserResult{}, notFoundError(opUpdateUser, entUser, arg.ID)
		}

		// else return internal/fk-violation error
		opErr := sqlError(
			opUpdateUser,
			opDetails{userID: arg.ID, entity: entUser},
			err,
		)

		return UpdateUserResult{}, opErr
	}

	// happy case
	if row.Updated {
		result := UpdateUserResult{
			ID:             row.ID,
			Username:       row.Username,
			Email:          row.Email,
			ProfileImgURL:  row.ProfileImgUrl,
			LastModifiedAt: row.LastModifiedAt,
		}

		return result, nil
	}

	// in case the user is soft-deleted return error
	if row.IsDeleted {
		opErr := newOpError(
			opUpdateUser,
			KindDeleted,
			entUser,
			fmt.Errorf("user with id %d is deleted and cannot be updated", arg.ID),
			withEntityID(arg.ID),
		)

		return UpdateUserResult{}, opErr
	}

	// guarding fallback
	opErr := newOpError(
		opUpdateUser,
		KindInternal,
		entUser,
		fmt.Errorf("failed to update user with id %d", arg.ID),
		withEntityID(arg.ID),
	)

	return UpdateUserResult{}, opErr
}
