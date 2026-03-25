package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const opGetUserByUsername = "get-user-by-username"

// GetUserByUsername retrieves the user with the provided username.
// Returns [KindNotFound] if the user does not exist, [KindDeleted] if the user
// has been soft-deleted, or [KindInternal] on database errors.
func (s *SQLStore) GetUserByUsername(ctx context.Context, username string) (User, error) {
	user, err := s.getUserByUsername(ctx, username)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			opErr := newOpError(
				opGetUserByUsername,
				KindNotFound,
				entUser,
				fmt.Errorf("user with name '%s' not found", username),
			)
			return User{}, opErr
		}

		return User{}, sqlError(
			opGetUserByUsername,
			opDetails{
				entity: entUser,
				input:  username,
			},
			err,
		)
	}

	if user.IsDeleted {
		opErr := newOpError(
			opGetUserByUsername,
			KindDeleted,
			entUser,
			fmt.Errorf("user with name '%s' is deleted", username),
		)

		return User{}, opErr
	}

	return user, nil
}
