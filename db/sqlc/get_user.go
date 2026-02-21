package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const opGetUser = "get-user"

// GetUser retrieves the user with the provided ID.
// Returns KindNotFound if the user does not exist, KindDeleted if the user
// has been soft-deleted, or KindInternal on database errors.
func (s *SQLStore) GetUser(ctx context.Context, userID int64) (User, error) {
	user, err := s.getUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, notFoundError(opGetUser, entUser, userID)
		}

		opErr := sqlError(
			opGetUser,
			opDetails{userID: userID, entity: entUser},
			err,
		)

		return User{}, opErr
	}

	// if user is soft-deleted return err
	if user.IsDeleted {
		opErr := newOpError(
			opGetUser,
			KindDeleted,
			entUser,
			fmt.Errorf("user with id %d is deleted", userID),
			withEntityID(userID),
		)

		return User{}, opErr
	}

	return user, nil
}
